package shorner

import (
	"context"
	"fmt"

	"github.com/joaovv-Vitor/Shorner-url/pkg/base62"
)

// Service implements the core use cases for URL shortening.
type Service struct {
	repo  Repository
	idGen IDGenerator
}

// NewService creates a new Service with the given dependencies.
func NewService(repo Repository, idGen IDGenerator) *Service {
	return &Service{
		repo:  repo,
		idGen: idGen,
	}
}

// ShortenInput represents the input for the Shorten use case.
type ShortenInput struct {
	OriginalURL string
}

// ShortenOutput represents the output of the Shorten use case.
type ShortenOutput struct {
	Code        string `json:"code"`
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}

// Shorten creates a shortened URL for the given original URL.
// If the URL has already been shortened, it returns the existing short code (idempotent).
func (s *Service) Shorten(ctx context.Context, input ShortenInput, baseURL string) (*ShortenOutput, error) {
	// Check if URL already exists (idempotency).
	existing, err := s.repo.FindByOriginalURL(ctx, input.OriginalURL)
	if err == nil && existing != nil {
		return &ShortenOutput{
			Code:        existing.Code,
			OriginalURL: existing.OriginalURL,
			ShortURL:    fmt.Sprintf("%s/%s", baseURL, existing.Code),
		}, nil
	}

	// Generate a new unique ID and encode it as Base62.
	id, err := s.idGen.NextID(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating id: %w", err)
	}

	code := base62.Encode(id)

	// Create the domain entity (validates the URL).
	shortURL, err := NewURL(code, input.OriginalURL)
	if err != nil {
		return nil, fmt.Errorf("creating url: %w", err)
	}

	// Persist.
	if err := s.repo.Save(ctx, shortURL); err != nil {
		return nil, fmt.Errorf("saving url: %w", err)
	}

	return &ShortenOutput{
		Code:        shortURL.Code,
		OriginalURL: shortURL.OriginalURL,
		ShortURL:    fmt.Sprintf("%s/%s", baseURL, shortURL.Code),
	}, nil
}

// Resolve retrieves the original URL for a given short code.
func (s *Service) Resolve(ctx context.Context, code string) (*URL, error) {
	u, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("resolving url: %w", err)
	}

	return u, nil
}
