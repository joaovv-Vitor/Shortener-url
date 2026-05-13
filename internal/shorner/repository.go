package shorner

import (
	"context"
	"sync"
	"sync/atomic"
)

// --- In-Memory Repository ---

// MemoryRepository is an in-memory implementation of Repository.
// It uses maps for O(1) lookups by code and by original URL.
// Thread-safe via sync.RWMutex.
type MemoryRepository struct {
	mu            sync.RWMutex
	byCode        map[string]*URL
	byOriginalURL map[string]*URL
}

// NewMemoryRepository creates a new empty MemoryRepository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byCode:        make(map[string]*URL),
		byOriginalURL: make(map[string]*URL),
	}
}

func (r *MemoryRepository) Save(_ context.Context, url *URL) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byCode[url.Code] = url
	r.byOriginalURL[url.OriginalURL] = url

	return nil
}

func (r *MemoryRepository) FindByCode(_ context.Context, code string) (*URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, ok := r.byCode[code]
	if !ok {
		return nil, ErrURLNotFound
	}

	return url, nil
}

func (r *MemoryRepository) FindByOriginalURL(_ context.Context, originalURL string) (*URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	url, ok := r.byOriginalURL[originalURL]
	if !ok {
		return nil, ErrURLNotFound
	}

	return url, nil
}

// --- In-Memory ID Generator ---

// MemoryIDGenerator is an in-memory implementation of IDGenerator.
// It uses an atomic counter for thread-safe ID generation.
// This will later be replaced by Redis INCR.
type MemoryIDGenerator struct {
	counter atomic.Uint64
}

// NewMemoryIDGenerator creates a new MemoryIDGenerator starting at 0.
func NewMemoryIDGenerator() *MemoryIDGenerator {
	return &MemoryIDGenerator{}
}

func (g *MemoryIDGenerator) NextID(_ context.Context) (uint64, error) {
	return g.counter.Add(1), nil
}
