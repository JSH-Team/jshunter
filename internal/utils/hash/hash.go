package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateMd5Hash(str string) string {
	hasher := md5.New()
	hasher.Write([]byte(str))
	return hex.EncodeToString(hasher.Sum(nil))
}

func GenerateSha256Hash(str string) string {
	hasher := sha256.New()
	hasher.Write([]byte(str))
	return hex.EncodeToString(hasher.Sum(nil))
}
