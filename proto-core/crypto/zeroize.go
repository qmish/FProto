package crypto

// Zeroize overwrites a byte slice with zeros to clear sensitive key material.
func Zeroize(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
