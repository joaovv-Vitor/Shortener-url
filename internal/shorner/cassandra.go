package shorner

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

// CassandraRepository is a Cassandra implementation of Repository.
// It uses two tables for efficient lookups:
//   - urls: keyed by short code (for redirects)
//   - urls_by_original: keyed by original URL (for idempotency)
type CassandraRepository struct {
	session *gocql.Session
}

// NewCassandraRepository creates a new CassandraRepository.
// The caller is responsible for closing the session when done.
func NewCassandraRepository(session *gocql.Session) *CassandraRepository {
	return &CassandraRepository{session: session}
}

func (r *CassandraRepository) Save(ctx context.Context, url *URL) error {
	// Use a logged batch to write to both tables atomically.
	batch := r.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)

	batch.Query(
		`INSERT INTO shorner.urls (code, original_url, created_at) VALUES (?, ?, ?)`,
		url.Code, url.OriginalURL, url.CreatedAt,
	)

	batch.Query(
		`INSERT INTO shorner.urls_by_original (original_url, code, created_at) VALUES (?, ?, ?)`,
		url.OriginalURL, url.Code, url.CreatedAt,
	)

	if err := r.session.ExecuteBatch(batch); err != nil {
		return fmt.Errorf("cassandra save: %w", err)
	}

	return nil
}

func (r *CassandraRepository) FindByCode(ctx context.Context, code string) (*URL, error) {
	var originalURL string
	var createdAt time.Time

	err := r.session.Query(
		`SELECT original_url, created_at FROM shorner.urls WHERE code = ?`, code,
	).WithContext(ctx).Scan(&originalURL, &createdAt)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("cassandra find by code: %w", err)
	}

	return &URL{
		Code:        code,
		OriginalURL: originalURL,
		CreatedAt:   createdAt,
	}, nil
}

func (r *CassandraRepository) FindByOriginalURL(ctx context.Context, originalURL string) (*URL, error) {
	var code string
	var createdAt time.Time

	err := r.session.Query(
		`SELECT code, created_at FROM shorner.urls_by_original WHERE original_url = ?`, originalURL,
	).WithContext(ctx).Scan(&code, &createdAt)

	if err != nil {
		if err == gocql.ErrNotFound {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("cassandra find by original url: %w", err)
	}

	return &URL{
		Code:        code,
		OriginalURL: originalURL,
		CreatedAt:   createdAt,
	}, nil
}

// --- Cassandra Connection Helper ---

// CassandraConfig holds the connection settings for Cassandra.
type CassandraConfig struct {
	Hosts       []string
	Keyspace    string
	Consistency gocql.Consistency
	Timeout     time.Duration
}

// NewCassandraSession creates a new gocql.Session with the given config.
func NewCassandraSession(cfg CassandraConfig) (*gocql.Session, error) {
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Keyspace = cfg.Keyspace
	cluster.Consistency = cfg.Consistency
	cluster.Timeout = cfg.Timeout
	cluster.ConnectTimeout = cfg.Timeout

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("cassandra connect: %w", err)
	}

	return session, nil
}
