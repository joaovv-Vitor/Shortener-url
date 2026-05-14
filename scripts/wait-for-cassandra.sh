#!/bin/sh
# wait-for-cassandra.sh — Waits for Cassandra to be ready and runs the init script.
# Used by the cassandra-init container in docker-compose.

set -e

HOST="${CASSANDRA_HOST:-cassandra}"
PORT="${CASSANDRA_PORT:-9042}"
CQL_FILE="${CQL_FILE:-/scripts/cassandra-init.cql}"

echo "Waiting for Cassandra at ${HOST}:${PORT}..."

until cqlsh "$HOST" "$PORT" -e "DESCRIBE KEYSPACES" > /dev/null 2>&1; do
    echo "  ...Cassandra not ready yet, retrying in 5s"
    sleep 5
done

echo "Cassandra is ready! Running init script..."
cqlsh "$HOST" "$PORT" -f "$CQL_FILE"
echo "Schema initialized successfully."
