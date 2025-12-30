-- Database Initialization Script for Metrics Collection System
--
-- This script creates the necessary tables and indexes for storing metrics.
-- It's designed to work with PostgreSQL and TimescaleDB.

-- Create the metrics table
CREATE TABLE IF NOT EXISTS metrics (
    time        TIMESTAMPTZ NOT NULL,
    name        TEXT NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    metric_type TEXT NOT NULL DEFAULT 'gauge',
    hostname    TEXT NOT NULL,
    labels      JSONB DEFAULT '{}',
    unit        TEXT DEFAULT ''
);

-- Create a hypertable for time-series data (TimescaleDB)
-- This command will fail gracefully if TimescaleDB is not installed
DO $$
BEGIN
    PERFORM create_hypertable('metrics', 'time', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'TimescaleDB not installed, using regular table';
END $$;

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_metrics_name_time ON metrics (name, time DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_hostname ON metrics (hostname);
CREATE INDEX IF NOT EXISTS idx_metrics_time ON metrics (time DESC);

-- Create index on labels for JSONB queries
CREATE INDEX IF NOT EXISTS idx_metrics_labels ON metrics USING GIN (labels);

-- Enable compression for old data (TimescaleDB)
DO $$
BEGIN
    ALTER TABLE metrics SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'name,hostname'
    );
    
    -- Add compression policy: compress data older than 7 days
    SELECT add_compression_policy('metrics', INTERVAL '7 days', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'TimescaleDB compression not available';
    WHEN OTHERS THEN
        RAISE NOTICE 'Could not set compression: %', SQLERRM;
END $$;

-- Add retention policy (TimescaleDB)
-- Keep data for 90 days by default
DO $$
BEGIN
    SELECT add_retention_policy('metrics', INTERVAL '90 days', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'TimescaleDB retention policy not available';
    WHEN OTHERS THEN
        RAISE NOTICE 'Could not set retention policy: %', SQLERRM;
END $$;

-- Create continuous aggregate for hourly metrics (TimescaleDB)
DO $$
BEGIN
    CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_hourly
    WITH (timescaledb.continuous) AS
    SELECT
        time_bucket('1 hour', time) AS bucket,
        name,
        hostname,
        AVG(value) AS avg_value,
        MIN(value) AS min_value,
        MAX(value) AS max_value,
        COUNT(*) AS sample_count
    FROM metrics
    GROUP BY bucket, name, hostname;

    -- Refresh policy for continuous aggregate
    SELECT add_continuous_aggregate_policy('metrics_hourly',
        start_offset => INTERVAL '3 hours',
        end_offset => INTERVAL '1 hour',
        schedule_interval => INTERVAL '1 hour',
        if_not_exists => TRUE
    );
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'TimescaleDB continuous aggregates not available';
    WHEN duplicate_table THEN
        RAISE NOTICE 'Continuous aggregate already exists';
    WHEN OTHERS THEN
        RAISE NOTICE 'Could not create continuous aggregate: %', SQLERRM;
END $$;

-- Grant permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON metrics TO metrics;
-- GRANT SELECT ON metrics_hourly TO metrics;

-- Create a function to query latest metrics for a host
CREATE OR REPLACE FUNCTION get_latest_metrics(
    p_hostname TEXT,
    p_limit INTEGER DEFAULT 100
)
RETURNS TABLE (
    time TIMESTAMPTZ,
    name TEXT,
    value DOUBLE PRECISION,
    labels JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT ON (m.name)
        m.time,
        m.name,
        m.value,
        m.labels
    FROM metrics m
    WHERE m.hostname = p_hostname
    ORDER BY m.name, m.time DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Create a function to get metric history
CREATE OR REPLACE FUNCTION get_metric_history(
    p_name TEXT,
    p_hostname TEXT DEFAULT NULL,
    p_start_time TIMESTAMPTZ DEFAULT NOW() - INTERVAL '1 hour',
    p_end_time TIMESTAMPTZ DEFAULT NOW()
)
RETURNS TABLE (
    time TIMESTAMPTZ,
    value DOUBLE PRECISION,
    hostname TEXT,
    labels JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT m.time, m.value, m.hostname, m.labels
    FROM metrics m
    WHERE m.name = p_name
      AND m.time >= p_start_time
      AND m.time <= p_end_time
      AND (p_hostname IS NULL OR m.hostname = p_hostname)
    ORDER BY m.time DESC;
END;
$$ LANGUAGE plpgsql;

-- Print success message
DO $$
BEGIN
    RAISE NOTICE 'Database initialization complete!';
END $$;
