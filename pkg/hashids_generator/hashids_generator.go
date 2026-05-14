// Package shortcode provides URL-safe short code generation using Hashids.
// It encodes numeric IDs into non-sequential, obfuscated strings using
// a secret salt and the Base62 alphabet.
package hashids_generator

import (
	"errors"
	"fmt"

	"github.com/speps/go-hashids/v2"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	ErrEmptyCode = errors.New("shortcode: empty code")
	ErrNoDecode  = errors.New("shortcode: no value decoded")
)

// Generator encodes and decodes numeric IDs to/from short codes
// using Hashids with a secret salt and Base62 alphabet.
type Generator struct {
	h         *hashids.HashID
	maxLength int
}

// NewGenerator creates a new Generator.
//   - salt: secret key that makes the encoding unique and unpredictable.
//   - minLength: minimum number of characters in the generated code.
//   - maxLength: maximum allowed characters (codes exceeding this are rejected).
func NewGenerator(salt string, minLength, maxLength int) (*Generator, error) {
	hd := hashids.NewData()
	hd.Salt = salt
	hd.MinLength = minLength
	hd.Alphabet = alphabet

	h, err := hashids.NewWithData(hd)
	if err != nil {
		return nil, fmt.Errorf("shortcode: initializing hashids: %w", err)
	}

	return &Generator{
		h:         h,
		maxLength: maxLength,
	}, nil
}

// Encode converts a uint64 ID into a short code string.
// Returns an error if the resulting code exceeds maxLength.
func (g *Generator) Encode(id uint64) (string, error) {
	code, err := g.h.EncodeInt64([]int64{int64(id)})
	if err != nil {
		return "", fmt.Errorf("shortcode: encoding id %d: %w", id, err)
	}

	if len(code) > g.maxLength {
		return "", fmt.Errorf("shortcode: code %q exceeds max length %d", code, g.maxLength)
	}

	return code, nil
}

// Decode converts a short code string back into the original uint64 ID.
func (g *Generator) Decode(code string) (uint64, error) {
	if code == "" {
		return 0, ErrEmptyCode
	}

	ids, err := g.h.DecodeInt64WithError(code)
	if err != nil {
		return 0, fmt.Errorf("shortcode: decoding %q: %w", code, err)
	}

	if len(ids) == 0 {
		return 0, ErrNoDecode
	}

	return uint64(ids[0]), nil
}
