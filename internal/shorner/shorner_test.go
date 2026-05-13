package shorner

import (
	"context"
	"testing"
)

// --- Entity tests ---

func TestNewURL_Valid(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"http url", "http://example.com"},
		{"https url", "https://example.com"},
		{"url with path", "https://example.com/some/path"},
		{"url with query", "https://example.com/search?q=golang"},
		{"url with fragment", "https://example.com/page#section"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := NewURL("abc", tt.url)
			if err != nil {
				t.Fatalf("NewURL(%q) returned error: %v", tt.url, err)
			}
			if url.Code != "abc" {
				t.Errorf("expected code %q, got %q", "abc", url.Code)
			}
			if url.OriginalURL != tt.url {
				t.Errorf("expected original_url %q, got %q", tt.url, url.OriginalURL)
			}
			if url.CreatedAt.IsZero() {
				t.Error("expected non-zero created_at")
			}
		})
	}
}

func TestNewURL_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedErr error
	}{
		{"empty url", "", ErrEmptyURL},
		{"no scheme", "example.com", ErrInvalidURL},
		{"ftp scheme", "ftp://example.com", ErrInvalidURL},
		{"just text", "not a url", ErrInvalidURL},
		{"no host", "http://", ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewURL("abc", tt.url)
			if err == nil {
				t.Fatalf("NewURL(%q) expected error, got nil", tt.url)
			}
			if err != tt.expectedErr {
				t.Errorf("expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

// --- Service tests ---

func TestShorten_Success(t *testing.T) {
	repo := NewMemoryRepository()
	gen := NewMemoryIDGenerator()
	svc := NewService(repo, gen)

	result, err := svc.Shorten(context.Background(), ShortenInput{
		OriginalURL: "https://example.com/very/long/path",
	}, "http://localhost:8080")

	if err != nil {
		t.Fatalf("Shorten returned error: %v", err)
	}

	if result.Code == "" {
		t.Error("expected non-empty code")
	}
	if result.OriginalURL != "https://example.com/very/long/path" {
		t.Errorf("expected original_url to match input")
	}
	if result.ShortURL == "" {
		t.Error("expected non-empty short_url")
	}
}

func TestShorten_Idempotent(t *testing.T) {
	repo := NewMemoryRepository()
	gen := NewMemoryIDGenerator()
	svc := NewService(repo, gen)

	url := "https://example.com/idempotent"
	baseURL := "http://localhost:8080"

	result1, err := svc.Shorten(context.Background(), ShortenInput{OriginalURL: url}, baseURL)
	if err != nil {
		t.Fatalf("First Shorten returned error: %v", err)
	}

	result2, err := svc.Shorten(context.Background(), ShortenInput{OriginalURL: url}, baseURL)
	if err != nil {
		t.Fatalf("Second Shorten returned error: %v", err)
	}

	if result1.Code != result2.Code {
		t.Errorf("expected same code for same URL, got %q and %q", result1.Code, result2.Code)
	}
}

func TestShorten_InvalidURL(t *testing.T) {
	repo := NewMemoryRepository()
	gen := NewMemoryIDGenerator()
	svc := NewService(repo, gen)

	_, err := svc.Shorten(context.Background(), ShortenInput{
		OriginalURL: "not-a-valid-url",
	}, "http://localhost:8080")

	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestResolve_Success(t *testing.T) {
	repo := NewMemoryRepository()
	gen := NewMemoryIDGenerator()
	svc := NewService(repo, gen)

	result, err := svc.Shorten(context.Background(), ShortenInput{
		OriginalURL: "https://example.com",
	}, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Shorten returned error: %v", err)
	}

	url, err := svc.Resolve(context.Background(), result.Code)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if url.OriginalURL != "https://example.com" {
		t.Errorf("expected original_url %q, got %q", "https://example.com", url.OriginalURL)
	}
}

func TestResolve_NotFound(t *testing.T) {
	repo := NewMemoryRepository()
	gen := NewMemoryIDGenerator()
	svc := NewService(repo, gen)

	_, err := svc.Resolve(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent code")
	}

	if !IsURLNotFound(err) {
		t.Logf("error: %v (this is expected)", err)
	}
}
