package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const rekeyInfo = "proto-rekey"
const rekeyLen = 32

// Rekey derives a new symmetric key from the current key using HKDF-SHA256.
// Returns the new 32-byte key. The caller is responsible for zeroizing the old key.
func Rekey(currentKey []byte) ([]byte, error) {
	if len(currentKey) == 0 {
		return nil, fmt.Errorf("rekey: empty current key")
	}

	r := hkdf.New(sha256.New, currentKey, nil, []byte(rekeyInfo))
	newKey := make([]byte, rekeyLen)
	if _, err := io.ReadFull(r, newKey); err != nil {
		return nil, fmt.Errorf("rekey hkdf: %w", err)
	}
	return newKey, nil
}
