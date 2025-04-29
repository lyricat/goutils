package attachment

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"strings"
)

func GuessExtByMimeType(mimeType string) string {
	if mimeType == "" {
		return ""
	}

	if strings.HasPrefix(mimeType, "image/jpeg") || strings.HasPrefix(mimeType, "image/jpg") {
		return ".jpg"
	} else if strings.HasPrefix(mimeType, "image/png") {
		return ".png"
	} else if strings.HasPrefix(mimeType, "image/gif") {
		return ".gif"
	} else if strings.HasPrefix(mimeType, "image/webp") {
		return ".webp"
	} else if strings.HasPrefix(mimeType, "image/svg") {
		return ".svg"
	}
	return ""
}

func getFileSha1Sum(file io.ReadSeeker) (string, error) {
	hasher := sha1.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	md5sum := hex.EncodeToString(hasher.Sum(nil))
	return md5sum, nil
}

func readerToReadSeeker(r io.Reader) (io.ReadSeeker, int64, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, err
	}
	return bytes.NewReader(buf), int64(len(buf)), nil
}
