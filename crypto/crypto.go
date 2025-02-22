package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/curve25519"
)

const NonceSize = 12

func GenKeyPair() (string, string) {
	privateKey := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(privateKey); err != nil {
		panic(err)
	}
	publicKey, _ := curve25519.X25519(privateKey, curve25519.Basepoint)
	return base64.RawStdEncoding.EncodeToString(privateKey), base64.RawStdEncoding.EncodeToString(publicKey)
}

func Encrypt(text, key string) (string, error) {
	buf, err := EncryptBytes([]byte(text), key)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func EncryptBytes(buf []byte, key string) ([]byte, error) {
	publicKey, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil || len(publicKey) != curve25519.PointSize {
		return nil, errors.New("invalid public key")
	}

	ephemeral := make([]byte, curve25519.ScalarSize)
	if _, err := rand.Read(ephemeral); err != nil {
		return nil, err
	}

	ePublicKey, err := curve25519.X25519(ephemeral, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := curve25519.X25519(ephemeral, publicKey)
	if err != nil {
		return nil, err
	}

	cipherText, nonce, err := AesGcmEncrypt(sharedSecret, buf)
	if err != nil {
		return nil, err
	}

	data := make([]byte, curve25519.PointSize+NonceSize+len(cipherText))
	copy(data, ePublicKey)
	copy(data[curve25519.PointSize:curve25519.PointSize+NonceSize], nonce)
	copy(data[curve25519.PointSize+NonceSize:], cipherText)
	out := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
	base64.RawStdEncoding.Encode(out, data)
	return out, nil
}

func Decrypt(text, key string) (string, error) {
	buf, err := DecryptBytes([]byte(text), key)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func DecryptBytes(buf []byte, key string) ([]byte, error) {
	data := make([]byte, base64.RawStdEncoding.DecodedLen(len(buf)))
	if _, err := base64.RawStdEncoding.Decode(data, buf); err != nil {
		return nil, err
	}

	privateKey, err := base64.RawStdEncoding.DecodeString(key)
	if err != nil || len(privateKey) != curve25519.ScalarSize {
		return nil, errors.New("invalid private key")
	}

	ePublicKey := data[:curve25519.PointSize]
	nonce := data[curve25519.PointSize : curve25519.PointSize+NonceSize]
	cipherText := data[curve25519.PointSize+NonceSize:]

	sharedSecret, err := curve25519.X25519(privateKey, ePublicKey)
	if err != nil {
		return nil, err
	}

	content, err := AesGcmDecrypt(sharedSecret, nonce, cipherText)
	return content, err
}

func AesGcmEncrypt(key, data []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	cipherText := aesgcm.Seal(nil, nonce, data, nil)

	return cipherText, nonce, nil
}

func AesGcmDecrypt(key, nonce, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesgcm.Open(nil, nonce, data, nil)
}
