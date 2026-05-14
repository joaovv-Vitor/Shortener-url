// Package shorner contains everything related to URL shortening:
// entities, interfaces (ports), use cases, handlers, and repositories.
package shorner

import (
	"context"
	"errors"
	"net/url"
	"time"
)

// --- Errors ---

var (
	ErrInvalidURL  = errors.New("invalid URL")
	ErrURLNotFound = errors.New("url not found")
	ErrEmptyURL    = errors.New("url cannot be empty")
)

// IsURLNotFound checks if an error wraps ErrURLNotFound.
func IsURLNotFound(err error) bool {
	return errors.Is(err, ErrURLNotFound)
}

// --- Entity ---

// URL represents a shortened URL entity.
type URL struct {
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewURL creates a new URL entity after validating the original URL.
func NewURL(code, originalURL string) (*URL, error) {
	if err := validateURL(originalURL); err != nil {
		return nil, err
	}

	return &URL{
		Code:        code,
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
	}, nil
}

// validateURL checks if the given string is a valid, absolute URL
// with an http or https scheme.
func validateURL(rawURL string) error {
	if rawURL == "" {
		return ErrEmptyURL
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}

	if parsed.Host == "" {
		return ErrInvalidURL
	}

	return nil
}

// --- Ports (interfaces) ---

// Repository defines the contract for URL persistence.
// Implementations can be in-memory, Cassandra, or any other storage.
type Repository interface {
	// Save persists a shortened URL.
	Save(ctx context.Context, url *URL) error

	// FindByCode retrieves a URL by its short code.
	FindByCode(ctx context.Context, code string) (*URL, error)

	// FindByOriginalURL retrieves a URL by the original URL.
	// This is used for idempotency — returning the same short code
	// if the URL has already been shortened.
	FindByOriginalURL(ctx context.Context, originalURL string) (*URL, error)
}

// IDGenerator defines the contract for generating unique numeric IDs.
// Implementations can be an in-memory counter or Redis INCR.
type IDGenerator interface {
	// NextID returns the next unique ID.
	NextID(ctx context.Context) (uint64, error)
}

// CodeEncoder defines the contract for encoding/decoding numeric IDs
// to/from short code strings. Implementations can use Base62, Hashids, etc.
type CodeEncoder interface {
	// Encode converts a numeric ID into a short code string.
	Encode(id uint64) (string, error)

	// Decode converts a short code string back into a numeric ID.
	Decode(code string) (uint64, error)
}
