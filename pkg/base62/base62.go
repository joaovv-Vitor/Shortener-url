// Package base62 provides encoding and decoding of uint64 values
// to/from Base62 strings (0-9, A-Z, a-z). This is used to convert
// numeric counter IDs into short, URL-safe codes.
package base62

import (
	"errors"
	"math"
	"strings"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const base = uint64(len(alphabet))

var ErrInvalidCharacter = errors.New("base62: invalid character")
var ErrOverflow = errors.New("base62: value overflows uint64")

// Encode converts a uint64 number into a Base62 string.
// Example: 1234 → "jU"
func Encode(num uint64) string {
	if num == 0 {
		return string(alphabet[0])
	}

	var builder strings.Builder
	for num > 0 {
		remainder := num % base
		builder.WriteByte(alphabet[remainder])
		num /= base
	}

	// Reverse the string since we built it in reverse order.
	bytes := []byte(builder.String())
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	return string(bytes)
}

// Decode converts a Base62 string back into a uint64 number.
// Returns an error if the string contains invalid characters or
// the result overflows uint64.
func Decode(s string) (uint64, error) {
	if len(s) == 0 {
		return 0, ErrInvalidCharacter
	}

	var num uint64
	for _, c := range s {
		idx := strings.IndexRune(alphabet, c)
		if idx < 0 {
			return 0, ErrInvalidCharacter
		}

		// Check for overflow before multiplying.
		if num > math.MaxUint64/base {
			return 0, ErrOverflow
		}
		num = num*base + uint64(idx)
	}

	return num, nil
}
