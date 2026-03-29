package crypto

// Zeroize overwrites a byte slice with zeros.
// Used to clear sensitive key material from memory.
func Zeroize(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
