package shorner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// --- Redis ID Generator ---

// RedisIDGenerator uses Redis INCR to generate globally unique, sequential IDs.
// This is atomic even with multiple web server instances, making it perfect
// for distributed short code generation.
type RedisIDGenerator struct {
	client *redis.Client
	key    string
}

// NewRedisIDGenerator creates a new RedisIDGenerator.
// The key parameter is the Redis key used for the INCR counter (e.g., "shorner:id_counter").
func NewRedisIDGenerator(client *redis.Client, key string) *RedisIDGenerator {
	return &RedisIDGenerator{
		client: client,
		key:    key,
	}
}

func (g *RedisIDGenerator) NextID(ctx context.Context) (uint64, error) {
	val, err := g.client.Incr(ctx, g.key).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incr: %w", err)
	}

	return uint64(val), nil
}

// --- Cached Repository (Decorator) ---

// CachedRepository wraps a primary Repository (Cassandra) with Redis cache.
// It implements the cache-aside pattern:
//   - Read: check cache → if miss, fetch from primary → populate cache
//   - Write: write to primary → populate cache (write-through)
type CachedRepository struct {
	primary Repository
	cache   *redis.Client
	ttl     time.Duration
}

// NewCachedRepository creates a new CachedRepository that wraps the primary
// repository with Redis caching. TTL controls how long cached entries live.
func NewCachedRepository(primary Repository, cache *redis.Client, ttl time.Duration) *CachedRepository {
	return &CachedRepository{
		primary: primary,
		cache:   cache,
		ttl:     ttl,
	}
}

// cacheKey returns the Redis key for a given short code.
func cacheKeyByCode(code string) string {
	return fmt.Sprintf("shorner:url:%s", code)
}

// cacheKeyByOriginal returns the Redis key for a given original URL.
func cacheKeyByOriginal(originalURL string) string {
	return fmt.Sprintf("shorner:original:%s", originalURL)
}

// cachedURL is the JSON representation stored in Redis cache.
type cachedURL struct {
	Code        string    `json:"code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *CachedRepository) Save(ctx context.Context, url *URL) error {
	// Write to primary storage first (Cassandra).
	if err := r.primary.Save(ctx, url); err != nil {
		return err
	}

	// Write-through: populate cache after successful write.
	r.cacheSet(ctx, url)

	return nil
}

func (r *CachedRepository) FindByCode(ctx context.Context, code string) (*URL, error) {
	// 1. Check cache.
	if url, err := r.cacheGetByCode(ctx, code); err == nil {
		return url, nil
	}

	// 2. Cache miss — fetch from primary.
	url, err := r.primary.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// 3. Populate cache for next time.
	r.cacheSet(ctx, url)

	return url, nil
}

func (r *CachedRepository) FindByOriginalURL(ctx context.Context, originalURL string) (*URL, error) {
	// 1. Check cache.
	if url, err := r.cacheGetByOriginal(ctx, originalURL); err == nil {
		return url, nil
	}

	// 2. Cache miss — fetch from primary.
	url, err := r.primary.FindByOriginalURL(ctx, originalURL)
	if err != nil {
		return nil, err
	}

	// 3. Populate cache for next time.
	r.cacheSet(ctx, url)

	return url, nil
}

// --- Cache helpers ---

// cacheSet stores a URL in both cache keys (by code and by original URL).
func (r *CachedRepository) cacheSet(ctx context.Context, url *URL) {
	data, err := json.Marshal(cachedURL{
		Code:        url.Code,
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt,
	})
	if err != nil {
		return // silently skip cache on marshal error
	}

	// Use pipeline for efficiency — both SET commands in one round trip.
	pipe := r.cache.Pipeline()
	pipe.Set(ctx, cacheKeyByCode(url.Code), data, r.ttl)
	pipe.Set(ctx, cacheKeyByOriginal(url.OriginalURL), data, r.ttl)
	_, _ = pipe.Exec(ctx) // best-effort caching
}

func (r *CachedRepository) cacheGetByCode(ctx context.Context, code string) (*URL, error) {
	return r.cacheGet(ctx, cacheKeyByCode(code))
}

func (r *CachedRepository) cacheGetByOriginal(ctx context.Context, originalURL string) (*URL, error) {
	return r.cacheGet(ctx, cacheKeyByOriginal(originalURL))
}

func (r *CachedRepository) cacheGet(ctx context.Context, key string) (*URL, error) {
	data, err := r.cache.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var cached cachedURL
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	return &URL{
		Code:        cached.Code,
		OriginalURL: cached.OriginalURL,
		CreatedAt:   cached.CreatedAt,
	}, nil
}

// --- Redis Connection Helper ---

// RedisConfig holds the connection settings for Redis.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// NewRedisClient creates a new redis.Client and verifies the connection with PING.
func NewRedisClient(ctx context.Context, cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connect: %w", err)
	}

	return client, nil
}
