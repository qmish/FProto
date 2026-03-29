package senderkeys

import (
	"crypto/hmac"
	"crypto/sha256"
)

// Ratchet derives the next chain_key and message_key from the current chain_key.
// next_chain_key = HMAC-SHA256(chain_key, "ratchet")
// message_key    = HMAC-SHA256(chain_key, "message")
func Ratchet(chainKey []byte) (nextChainKey, messageKey []byte) {
	nextChainKey = hmacSHA256(chainKey, []byte("ratchet"))
	messageKey = hmacSHA256(chainKey, []byte("message"))
	return
}

// RatchetN advances the chain N steps, returning the final chain_key and all message_keys.
func RatchetN(chainKey []byte, n int) (finalChainKey []byte, messageKeys [][]byte) {
	current := chainKey
	messageKeys = make([][]byte, n)
	for i := 0; i < n; i++ {
		next, mk := Ratchet(current)
		messageKeys[i] = mk
		current = next
	}
	return current, messageKeys
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
