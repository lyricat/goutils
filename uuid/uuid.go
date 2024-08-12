package uuid

import (
	"crypto/md5"
	"io"

	"github.com/gofrs/uuid"
)

func New() string {
	return uuid.Must(uuid.NewV4()).String()
}

func IsUUID(id string) bool {
	_, err := FromString(id)
	return err == nil
}

func MD5(input string) string {
	h := md5.New()
	io.WriteString(h, input)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return uuid.FromBytesOrNil(sum).String()
}

func Modify(id, modifier string) string {
	ns, err := uuid.FromString(id)
	if err != nil {
		panic(err)
	}
	return uuid.NewV5(ns, modifier).String()
}

func FromString(id string) (uuid.UUID, error) {
	return uuid.FromString(id)
}

func FromUint64(id uint64) (uuid.UUID, error) {
	// uint64 to bytes
	input := make([]byte, 16)
	for i := 0; i < 8; i++ {
		input[i] = byte(id >> (i * 8))
	}
	return uuid.FromBytes(input)
}

func IsNil(id string) bool {
	uid, err := FromString(id)
	return err != nil || uid == uuid.Nil
}
