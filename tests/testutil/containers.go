package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// RedisContainer represents a test Redis instance
type RedisContainer struct {
	Host string
	Port string
	Client *redis.Client
	cleanup func()
}

// PostgresContainer represents a test PostgreSQL instance
type PostgresContainer struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
	DB       *gorm.DB
	cleanup  func()
}

// StartRedis starts a Redis container for testing
// Note: This is a placeholder until Testcontainers network issues are resolved
func StartRedis(t *testing.T) *RedisContainer {
	t.Helper()

	// For now, use local Redis or skip if not available
	// TODO: Replace with Testcontainers once network is available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing:", err)
	}

	// Flush test database
	require.NoError(t, client.FlushDB(context.Background()).Err())

	return &RedisContainer{
		Host:   "localhost",
		Port:   "6379",
		Client: client,
		cleanup: func() {
			if err := client.FlushDB(context.Background()).Err(); err != nil {
				slog.Error("failed to flush redis database during cleanup", "error", err)
			}
			if err := client.Close(); err != nil {
				slog.Error("failed to close redis client", "error", err)
			}
		},
	}
}

// Close cleans up the Redis container
func (r *RedisContainer) Close() {
	if r.cleanup != nil {
		r.cleanup()
	}
}

// StartPostgres starts a PostgreSQL container for testing
// Note: This is a placeholder until Testcontainers network issues are resolved
func StartPostgres(t *testing.T) *PostgresContainer {
	t.Helper()

	// For now, use local PostgreSQL or skip if not available
	// TODO: Replace with Testcontainers once network is available
	dsn := "host=localhost user=quotient_test password=test123 dbname=quotient_test port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skip("PostgreSQL not available for testing:", err)
	}

	// Run migrations to ensure schema exists
	// Note: AutoMigrate will create tables if they don't exist
	db.Exec(`
		CREATE TABLE IF NOT EXISTS round_schemas (
			id SERIAL PRIMARY KEY,
			start_time TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS service_check_schemas (
			id SERIAL PRIMARY KEY,
			team_id INTEGER,
			round_id INTEGER,
			service_name TEXT,
			points INTEGER,
			result BOOLEAN,
			error TEXT,
			debug TEXT
		);
		CREATE TABLE IF NOT EXISTS sla_schemas (
			id SERIAL PRIMARY KEY,
			team_id INTEGER,
			service_name TEXT,
			round_id INTEGER,
			penalty INTEGER
		);
		CREATE TABLE IF NOT EXISTS team_schemas (
			id SERIAL PRIMARY KEY,
			name TEXT
		);
		CREATE TABLE IF NOT EXISTS uptime_schemas (
			id SERIAL PRIMARY KEY,
			team_id INTEGER,
			service_name TEXT,
			passed_checks INTEGER,
			total_checks INTEGER
		);
	`)

	return &PostgresContainer{
		Host:     "localhost",
		Port:     "5432",
		Database: "quotient_test",
		Username: "quotient_test",
		Password: "test123",
		DB:       db,
		cleanup: func() {
			// Clean up test data
			db.Exec("TRUNCATE TABLE round_schemas CASCADE")
			db.Exec("TRUNCATE TABLE service_check_schemas CASCADE")
			db.Exec("TRUNCATE TABLE sla_schemas CASCADE")
			db.Exec("TRUNCATE TABLE team_schemas CASCADE")
			db.Exec("TRUNCATE TABLE uptime_schemas CASCADE")

			sqlDB, _ := db.DB()
			if sqlDB != nil {
				if err := sqlDB.Close(); err != nil {
					slog.Error("failed to close postgres database", "error", err)
				}
			}
		},
	}
}

// Close cleans up the PostgreSQL container
func (p *PostgresContainer) Close() {
	if p.cleanup != nil {
		p.cleanup()
	}
}

// ConnectionString returns the PostgreSQL connection string
func (p *PostgresContainer) ConnectionString() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		p.Host, p.Username, p.Password, p.Database, p.Port)
}

// WaitForReady waits for a service to become ready
func WaitForReady(ctx context.Context, checkFn func() error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := checkFn(); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("service not ready after %v", timeout)
}
