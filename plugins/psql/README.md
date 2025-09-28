# PostgreSQL Plugin

A ZMS observer plugin that stores Zabbix export data in a PostgreSQL database.

## Features

- Stores history data in PostgreSQL database
- Configurable connection pooling settings
- Transaction-based writes for data consistency
- Supports tag-based filtering
- PostgreSQL connection metrics
- Prometheus metrics integration

## Configuration

Add to your ZMS configuration:

```yaml
targets:
  - name: "postgresql-storage"
    type: "postgresql"
    connection: "postgres://user:password@localhost/dbname?sslmode=disable"
    options:
      max_conn: "10"           # Maximum open connections
      max_idle: "5"            # Maximum idle connections
      max_conn_time: "1h"      # Maximum connection lifetime
      max_idle_time: "10m"     # Maximum idle time
    exports:
      - "history"
```

## Connection String

The connection string should be a PostgreSQL connection URL:
- Format: `postgres://[user[:password]@][netloc][:port][/dbname][?param1=value1&...]`
- Examples:
  - `postgres://user:pass@localhost/mydb?sslmode=disable`
  - `postgres://user@localhost:5432/mydb?sslmode=require`
  - `postgres://localhost/mydb?user=postgres&password=secret`

## Database Schema

The plugin expects a table with the following structure:

```sql
CREATE SCHEMA IF NOT EXISTS performance;

CREATE TABLE performance.messages (
    id SERIAL PRIMARY KEY,
    tagname VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    quality BOOLEAN NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    servertimestamp TIMESTAMP NOT NULL
);

-- Optional indexes for better performance
CREATE INDEX idx_messages_tagname ON performance.messages (tagname);
CREATE INDEX idx_messages_timestamp ON performance.messages (timestamp);
```

## Connection Options

- `max_conn`: Maximum number of open connections to the database (default: unlimited)
- `max_idle`: Maximum number of idle connections in the pool (default: 2)
- `max_conn_time`: Maximum amount of time a connection may be reused (e.g., "1h", "30m")
- `max_idle_time`: Maximum amount of time a connection may be idle (e.g., "10m", "5s")

## Data Format

### History Data
Each history record is stored as:
- **tagname**: `{hostname}.{hostname}.{item_name}` (duplicated hostname format)
- **value**: The metric value as text
- **quality**: Always `true`
- **timestamp**: Converted from Unix timestamp to PostgreSQL timestamp
- **servertimestamp**: Same as timestamp

## Connection Metrics

The plugin exposes PostgreSQL connection pool metrics:

- `zms_psql_connection_stats{conn="idle"}` - Number of idle connections
- `zms_psql_connection_stats{conn="max"}` - Maximum number of connections
- `zms_psql_connection_stats{conn="used"}` - Number of connections currently in use

## Building

```bash
go build -buildmode=plugin -o psql.so ./plugins/psql
```

## Dependencies

- `github.com/lib/pq` - PostgreSQL driver for Go
- `github.com/prometheus/client_golang/prometheus` - Prometheus metrics
- Standard Go libraries

## Performance Considerations

- Uses prepared statements for better performance
- All inserts are wrapped in transactions
- Connection pooling helps manage database load
- Consider adding database indexes on frequently queried columns

## Error Handling

- Failed transactions are rolled back automatically
- Connection failures are logged with detailed error information
- Failed operations are tracked in Prometheus metrics
- Database connection is validated on initialization

## PostgreSQL Setup

Ensure your PostgreSQL instance is configured:

```sql
-- Create database and user
CREATE DATABASE zms_data;
CREATE USER zms_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE zms_data TO zms_user;

-- Connect to the zms_data database and create schema
\c zms_data;
CREATE SCHEMA performance;
GRANT ALL ON SCHEMA performance TO zms_user;
```