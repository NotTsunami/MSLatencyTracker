// Package store is the data-access layer. It owns an in-memory cache of the
// latest reading per channel (for instant reads) and the SQL queries against
// the latency_history table.
package store

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"mslatencytracker/config"
)

// LatencyReading is a single latest measurement for one channel.
type LatencyReading struct {
	LatencyMs float64 `json:"latencyMs"`
	Timestamp int64   `json:"timestamp"`
}

// ChannelLatency is a reading plus its channel number.
type ChannelLatency struct {
	Channel   int     `json:"channel"`
	LatencyMs float64 `json:"latencyMs"`
	Timestamp int64   `json:"timestamp"`
}

// LatencyDataPoint is one point in a history time series.
type LatencyDataPoint struct {
	Timestamp int64   `json:"timestamp"`
	LatencyMs float64 `json:"latencyMs"`
}

// AverageResult is the computed average over a time window.
type AverageResult struct {
	AverageMs   float64 `json:"averageMs"`
	SampleCount int     `json:"sampleCount"`
}

// Store bundles the database handle with the in-memory latest-reading cache.
type Store struct {
	db *sql.DB

	mu     sync.RWMutex
	latest map[string]LatencyReading
}

// New constructs a Store.
func New(db *sql.DB) *Store {
	return &Store{
		db:     db,
		latest: make(map[string]LatencyReading),
	}
}

// key builds the cache key for a world+channel pair, e.g. "Kronos:7".
func key(w config.World, channel int) string {
	return fmt.Sprintf("%s:%d", w, channel)
}

// RecordLatency updates the in-memory cache and appends a row to the history
// table. A latencyMs of -1 means the server was unreachable / timed out.
func (s *Store) RecordLatency(w config.World, channel int, latencyMs float64, timestamp int64) {
	// Release the lock before the slower database call below.
	s.mu.Lock()
	s.latest[key(w, channel)] = LatencyReading{LatencyMs: latencyMs, Timestamp: timestamp}
	s.mu.Unlock()

	_, err := s.db.Exec(
		`INSERT INTO latency_history (world, channel, recorded_at, latency_ms)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (world, channel, recorded_at) DO NOTHING`,
		string(w), channel, timestamp, latencyMs,
	)
	if err != nil {
		// Log and move on: a single failed insert shouldn't take down the
		// whole ping cycle.
		log.Printf("Failed to persist latency for %s ch%d: %v", w, channel, err)
	}
}

// GetAllLatest returns the cached latest reading for every channel of a world
// that has reported at least once, in ascending channel order.
func (s *Store) GetAllLatest(w config.World) []ChannelLatency {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start from an empty (non-nil) slice so the JSON output is `[]`, not
	// `null`, when there are no readings yet.
	results := []ChannelLatency{}

	ips := config.Servers[w]
	for i := range ips {
		channel := i + 1
		if reading, ok := s.latest[key(w, channel)]; ok {
			results = append(results, ChannelLatency{
				Channel:   channel,
				LatencyMs: reading.LatencyMs,
				Timestamp: reading.Timestamp,
			})
		}
	}
	return results
}

// GetLatest returns the cached reading for one channel, and whether it exists.
func (s *Store) GetLatest(w config.World, channel int) (LatencyReading, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reading, ok := s.latest[key(w, channel)]
	return reading, ok
}

// GetAverage computes the average latency over the given period. Splitting
// "no data" (found=false) from "query failed" (err != nil) lets the caller
// return 404 vs 500 appropriately. Unreachable readings (-1) are excluded.
func (s *Store) GetAverage(w config.World, channel int, period time.Duration) (AverageResult, bool, error) {
	since := time.Now().Add(-period).UnixMilli()

	// AVG over zero rows returns SQL NULL, hence the sql.NullFloat64.
	var avg sql.NullFloat64
	var count int

	err := s.db.QueryRow(
		`SELECT AVG(latency_ms), COUNT(*)
		 FROM latency_history
		 WHERE world = $1 AND channel = $2 AND recorded_at >= $3 AND latency_ms >= 0`,
		string(w), channel, since,
	).Scan(&avg, &count)
	if err != nil {
		return AverageResult{}, false, err
	}

	if count == 0 || !avg.Valid {
		return AverageResult{}, false, nil
	}
	return AverageResult{AverageMs: avg.Float64, SampleCount: count}, true, nil
}

// GetHistory returns every data point for a channel within the period,
// ordered oldest-first (ready for a time-series chart).
func (s *Store) GetHistory(w config.World, channel int, period time.Duration) ([]LatencyDataPoint, error) {
	since := time.Now().Add(-period).UnixMilli()

	rows, err := s.db.Query(
		`SELECT recorded_at, latency_ms
		 FROM latency_history
		 WHERE world = $1 AND channel = $2 AND recorded_at >= $3
		 ORDER BY recorded_at ASC`,
		string(w), channel, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := []LatencyDataPoint{}
	for rows.Next() {
		var p LatencyDataPoint
		if err := rows.Scan(&p.Timestamp, &p.LatencyMs); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
