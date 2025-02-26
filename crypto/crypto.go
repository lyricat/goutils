package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	// NonceSize for AES-GCM.
	NonceSize   = 12
	versionByte = 1
)

// GenKeyPair creates a random X25519 key pair. Returns (privateKey, publicKey) in base64.
func GenKeyPair() (string, string) {
	priv := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(priv); err != nil {
		panic(err)
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		panic(err)
	}
	return base64.RawStdEncoding.EncodeToString(priv),
		base64.RawStdEncoding.EncodeToString(pub)
}

func Encrypt(plaintext, base64ReceiverPub string) (string, error) {
	out, err := EncryptBytes([]byte(plaintext), base64ReceiverPub)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// EncryptBytes encrypts using an ephemeral-to-static X25519 + HKDF + AES-GCM scheme.
// It always produces new-format ciphertext, which is:
//
//	[1-byte versionByte=1] || [32-byte ephemeralPubKey] || [12-byte nonce] || [AES-GCM ciphertext...]
//
// in which,
// - versionByte is the version of this format.
// - ephemeralPubKey is included as AES-GCM AAD (so it's integrity-protected).
// - The rawData is then base64.RawStdEncoded.
// - The derived AES key is 32 bytes from HKDF(sha256) of the X25519 shared secret.
func EncryptBytes(plaintext []byte, base64ReceiverPub string) ([]byte, error) {
	// decode receiver's public key
	receiverPub, err := base64.RawStdEncoding.DecodeString(base64ReceiverPub)
	if err != nil || len(receiverPub) != curve25519.PointSize {
		return nil, errors.New("invalid receiver public key")
	}

	// generate ephemeral private key and public key
	ephPriv := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(ephPriv); err != nil {
		return nil, err
	}
	ephPub, err := curve25519.X25519(ephPriv, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}

	// ECDH => raw shared secret
	sharedSecret, err := curve25519.X25519(ephPriv, receiverPub)
	if err != nil {
		return nil, err
	}
	// check for all-zero
	if subtle.ConstantTimeCompare(sharedSecret, make([]byte, 32)) == 1 {
		return nil, errors.New("invalid ECDH result (all-zero key)")
	}

	// HKDF derive AES-256 key
	aesKey, err := deriveAesKey(sharedSecret, ephPub)
	if err != nil {
		return nil, err
	}

	// do AES-GCM encrypt, ephemeralPub as AAD
	cipherText, nonce, err := aesGcmEncrypt(aesKey, plaintext, ephPub)
	if err != nil {
		return nil, err
	}

	// generate output => versionByte + ephPub(32) + nonce(12) + cipherText
	outLen := 1 + len(ephPub) + NonceSize + len(cipherText)
	outBuf := make([]byte, outLen)
	outBuf[0] = versionByte
	copy(outBuf[1:1+32], ephPub)
	copy(outBuf[1+32:1+32+NonceSize], nonce)
	copy(outBuf[1+32+NonceSize:], cipherText)

	encoded := make([]byte, base64.RawStdEncoding.EncodedLen(outLen))
	base64.RawStdEncoding.Encode(encoded, outBuf)
	return encoded, nil
}

func Decrypt(ciphertext, base64ReceiverPriv string) (string, error) {
	out, err := DecryptBytes([]byte(ciphertext), base64ReceiverPriv)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// DecryptBytes will handle:
//   - **New format**: with the leading [1-byte versionByte=1].
//   - **Old format**: no version byte, ephemeralPubKey(32) + nonce(12) + raw AES-GCM ciphertext.
//
// In the new format, we do HKDF on the X25519 shared secret, then use ephemeralPub as AAD.
// In the old format, we do NOT do HKDF nor ephemeralPub as AAD. We also do NOT store a version byte.
func DecryptBytes(cipherBuf []byte, base64ReceiverPriv string) ([]byte, error) {
	rawData := make([]byte, base64.RawStdEncoding.DecodedLen(len(cipherBuf)))
	n, err := base64.RawStdEncoding.Decode(rawData, cipherBuf)
	if err != nil {
		return nil, err
	}
	rawData = rawData[:n]

	// decode the receiver's private key
	receiverPriv, err := base64.RawStdEncoding.DecodeString(base64ReceiverPriv)
	if err != nil || len(receiverPriv) != curve25519.ScalarSize {
		return nil, errors.New("invalid receiver private key")
	}

	if len(rawData) < 32+NonceSize {
		return nil, errors.New("ciphertext too short")
	}

	if rawData[0] == versionByte && len(rawData) >= 1+32+NonceSize {
		// NEW FORMAT
		// Format: versionByte(1) + ephemeralPubKey(32) + nonce(12) + AES-GCM ciphertext
		ephemeralPub := rawData[1 : 1+32]
		nonce := rawData[1+32 : 1+32+NonceSize]
		cText := rawData[1+32+NonceSize:]

		// X25519 => sharedSecret
		sharedSecret, err := curve25519.X25519(receiverPriv, ephemeralPub)
		if err != nil {
			return nil, err
		}

		// Check for zero key
		if subtle.ConstantTimeCompare(sharedSecret, make([]byte, 32)) == 1 {
			return nil, errors.New("invalid ECDH result (all-zero key)")
		}

		// Derive final AES key from HKDF
		aesKey, err := deriveAesKey(sharedSecret, ephemeralPub)
		if err != nil {
			return nil, err
		}

		// AES-GCM decrypt with ephemeralPub as AAD
		return aesGcmDecrypt(aesKey, nonce, cText, ephemeralPub)

	} else {
		// OLD (LEGACY) FORMAT
		// Format: ephemeralPubKey(32) + nonce(12) + AES-GCM ciphertext
		if len(rawData) < 32+NonceSize {
			return nil, errors.New("ciphertext too short for old format")
		}

		ephemeralPub := rawData[0:32]
		nonce := rawData[32 : 32+NonceSize]
		cText := rawData[32+NonceSize:]

		// Perform ECDH
		sharedSecret, err := curve25519.X25519(receiverPriv, ephemeralPub)
		if err != nil {
			return nil, err
		}
		// In the old format, the old code used the raw ECDH output as AES key.
		aesKey := sharedSecret

		// Old code did not use ephemeralPubKey as AAD => pass nil as AAD
		return oldAesGcmDecrypt(aesKey, nonce, cText)
	}
}

// new-format encryption with ephemeralPub as AAD
func aesGcmEncrypt(key, plaintext, aad []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// new-format decryption with ephemeralPub as AAD
func aesGcmDecrypt(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, aad)
}

// old-format decryption: no AAD, raw ECDH key
func oldAesGcmDecrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// HKDF derive AES key from raw ECDH output (new format only).
func deriveAesKey(sharedSecret, ephemeralPub []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, sharedSecret, nil, ephemeralPub)
	aesKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(h, aesKey); err != nil {
		return nil, err
	}
	return aesKey, nil
}
