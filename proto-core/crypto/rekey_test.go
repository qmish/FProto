package crypto

import (
	"bytes"
	"testing"
)

func TestRekey(t *testing.T) {
	key := randomKey()
	original := make([]byte, 32)
	copy(original, key)

	newKey, err := Rekey(key)
	if err != nil {
		t.Fatalf("rekey: %v", err)
	}

	if len(newKey) != 32 {
		t.Fatalf("new key length: expected 32, got %d", len(newKey))
	}

	if bytes.Equal(key, newKey) {
		t.Fatal("new key should differ from original")
	}
}

func TestRekey_Deterministic(t *testing.T) {
	key := randomKey()

	r1, _ := Rekey(key)
	r2, _ := Rekey(key)

	if !bytes.Equal(r1, r2) {
		t.Fatal("rekey should be deterministic for same input")
	}
}

func TestRekey_EmptyKeyFails(t *testing.T) {
	_, err := Rekey(nil)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestRekey_ChainedRekey(t *testing.T) {
	key := randomKey()
	seen := make(map[string]bool)
	seen[string(key)] = true

	current := key
	for i := 0; i < 100; i++ {
		next, err := Rekey(current)
		if err != nil {
			t.Fatalf("chained rekey %d: %v", i, err)
		}
		if seen[string(next)] {
			t.Fatalf("key collision at iteration %d", i)
		}
		seen[string(next)] = true
		current = next
	}
}

func TestRekey_WithZeroize(t *testing.T) {
	key := randomKey()
	oldKey := make([]byte, 32)
	copy(oldKey, key)

	newKey, _ := Rekey(key)
	Zeroize(key)

	for _, b := range key {
		if b != 0 {
			t.Fatal("old key should be zeroed")
		}
	}

	if len(newKey) != 32 {
		t.Fatal("new key should be valid")
	}
}

func TestZeroize(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	Zeroize(data)
	for i, b := range data {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: %d", i, b)
		}
	}
}

func TestZeroize_Empty(t *testing.T) {
	Zeroize(nil)
	Zeroize([]byte{})
}
