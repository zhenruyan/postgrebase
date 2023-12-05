package tygoja

import (
	mathRand "math/rand"
	"time"
)

const defaultRandomAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func init() {
	mathRand.Seed(time.Now().UnixNano())
}

// PseudorandomString generates a pseudorandom string from the default
// alphabet with the specified length.
func PseudorandomString(length int) string {
	return PseudorandomStringWithAlphabet(length, defaultRandomAlphabet)
}

// PseudorandomStringWithAlphabet generates a pseudorandom string
// with the specified length and characters set.
func PseudorandomStringWithAlphabet(length int, alphabet string) string {
	b := make([]byte, length)
	max := len(alphabet)

	for i := range b {
		b[i] = alphabet[mathRand.Intn(max)]
	}

	return string(b)
}
