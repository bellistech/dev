// Package storage provides storage implementations for metrics.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Storage defines the interface for metric storage.
type Storage interface {
	// Store stores a batch of metrics.
	Store(ctx context.Context, metrics []metrics.Metric) error
	// Query retrieves metrics matching the given criteria.
	Query(ctx context.Context, name string, start, end time.Time, labels map[string]string) ([]metrics.Metric, error)
	// Ping checks if the storage is available.
	Ping(ctx context.Context) error
	// Close closes the storage connection.
	Close() error
}

// PostgresStorage implements Storage using PostgreSQL/TimescaleDB.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage.
func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

// Store stores a batch of metrics.
func (s *PostgresStorage) Store(ctx context.Context, metricsList []metrics.Metric) error {
	if len(metricsList) == 0 {
		return nil
	}

	// Use a transaction for batch insert
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the insert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics (time, name, value, metric_type, hostname, labels, unit)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each metric
	for _, m := range metricsList {
		labels := formatLabels(m.Labels)
		_, err := stmt.ExecContext(ctx,
			m.Timestamp,
			m.Name,
			m.Value,
			m.Type.String(),
			m.Hostname,
			labels,
			m.Unit,
		)
		if err != nil {
			log.Printf("Failed to insert metric %s: %v", m.Name, err)
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Query retrieves metrics matching the given criteria.
func (s *PostgresStorage) Query(ctx context.Context, name string, start, end time.Time, labels map[string]string) ([]metrics.Metric, error) {
	query := `
		SELECT time, name, value, metric_type, hostname, labels, unit
		FROM metrics
		WHERE name = $1 AND time >= $2 AND time <= $3
	`
	args := []interface{}{name, start, end}

	// Add label filters
	if len(labels) > 0 {
		for k, v := range labels {
			query += fmt.Sprintf(" AND labels->>'%s' = $%d", k, len(args)+1)
			args = append(args, v)
		}
	}

	query += " ORDER BY time DESC LIMIT 10000"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var result []metrics.Metric
	for rows.Next() {
		var m metrics.Metric
		var metricType string
		var labelsJSON sql.NullString
		var unit sql.NullString

		err := rows.Scan(&m.Timestamp, &m.Name, &m.Value, &metricType, &m.Hostname, &labelsJSON, &unit)
		if err != nil {
			continue
		}

		m.Type = parseMetricType(metricType)
		m.Labels = parseLabels(labelsJSON.String)
		m.Unit = unit.String

		result = append(result, m)
	}

	return result, rows.Err()
}

// Ping checks if the database is available.
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// formatLabels converts labels map to JSON string.
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}

	pairs := make([]string, 0, len(labels))
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf(`"%s":"%s"`, k, v))
	}
	return "{" + strings.Join(pairs, ",") + "}"
}

// parseLabels converts JSON string to labels map.
func parseLabels(s string) map[string]string {
	// Simple parsing - in production use proper JSON parsing
	labels := make(map[string]string)
	if s == "" || s == "{}" {
		return labels
	}

	// Remove braces
	s = strings.Trim(s, "{}")
	if s == "" {
		return labels
	}

	// Split by comma and parse key-value pairs
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			key := strings.Trim(kv[0], `"`)
			value := strings.Trim(kv[1], `"`)
			labels[key] = value
		}
	}

	return labels
}

// parseMetricType converts string to MetricType.
func parseMetricType(s string) metrics.MetricType {
	switch s {
	case "gauge":
		return metrics.MetricTypeGauge
	case "counter":
		return metrics.MetricTypeCounter
	case "summary":
		return metrics.MetricTypeSummary
	case "histogram":
		return metrics.MetricTypeHistogram
	default:
		return metrics.MetricTypeGauge
	}
}
