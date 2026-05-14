-- Shorner URL Shortener - Cassandra Schema
-- Run: cqlsh -f scripts/cassandra-init.cql

CREATE KEYSPACE IF NOT EXISTS shorner
WITH replication = {
    'class': 'SimpleStrategy',
    'replication_factor': 1
};

-- Primary table: lookup by short code (used for redirects).
CREATE TABLE IF NOT EXISTS shorner.urls (
    code       TEXT,
    original_url TEXT,
    created_at   TIMESTAMP,
    PRIMARY KEY (code)
);

-- Reverse lookup table: find by original URL (used for idempotency).
-- Avoids expensive secondary indexes in Cassandra.
CREATE TABLE IF NOT EXISTS shorner.urls_by_original (
    original_url TEXT,
    code         TEXT,
    created_at   TIMESTAMP,
    PRIMARY KEY (original_url)
);
