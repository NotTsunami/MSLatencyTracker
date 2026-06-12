// Package pinger runs the background loop that pings every configured channel
// IP on an interval and records the round-trip latency.
package pinger

import (
	"context"
	"log"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"

	"mslatencytracker/config"
	"mslatencytracker/store"
)

// pingOnce pings a single IP once and returns the round-trip time in
// milliseconds.
//
// On any failure it returns -1 (the agreed "unreachable" sentinel) so the API
// can still report that the channel was checked but did not respond.
func pingOnce(w config.World, channel int, ip string, timeout time.Duration) float64 {
	latencyMs := -1.0

	pinger, err := probing.NewPinger(ip)
	if err != nil {
		log.Printf("Ping setup failed for %s ch%d (%s): %v", w, channel, ip, err)
		return latencyMs
	}

	pinger.Count = 1
	pinger.Timeout = timeout
	// Raw ICMP sockets: needs NET_RAW on Linux (granted in docker-compose.yml).
	// Unprivileged UDP ping is disallowed by many hosts.
	pinger.SetPrivileged(true)

	if err := pinger.Run(); err != nil {
		log.Printf("Ping failed for %s ch%d (%s): %v", w, channel, ip, err)
	} else {
		stats := pinger.Statistics()
		if stats.PacketsRecv > 0 {
			latencyMs = float64(stats.AvgRtt.Microseconds()) / 1000.0
		}
	}

	return latencyMs
}

// pingAllChannels pings every configured channel of every world in parallel,
// waits for all of them to finish, then records the whole cycle as one batch
// (a single multi-row INSERT).
func pingAllChannels(s *store.Store, timeout time.Duration) {
	// One shared timestamp for the whole cycle, so all readings from this
	// pass line up on the same x-axis value in charts.
	timestamp := time.Now().UnixMilli()

	// One pre-allocated slot per channel; each goroutine writes only its own
	// slot, so the WaitGroup is the only synchronization needed.
	samples := []store.LatencySample{}
	for _, w := range config.WorldOrder {
		for i := range config.Servers[w] {
			samples = append(samples, store.LatencySample{World: w, Channel: i + 1, Timestamp: timestamp})
		}
	}

	var wg sync.WaitGroup
	for i := range samples {
		wg.Add(1)
		go func(sample *store.LatencySample) {
			defer wg.Done()
			ip := config.Servers[sample.World][sample.Channel-1]
			sample.LatencyMs = pingOnce(sample.World, sample.Channel, ip, timeout)
		}(&samples[i])
	}
	wg.Wait()

	s.RecordLatencyBatch(samples)
	log.Printf("Ping cycle complete. Pinged %d servers.", len(samples))
}

// Start launches the ping worker in the background and returns immediately.
// The worker stops when ctx is cancelled.
func Start(ctx context.Context, s *store.Store, interval, timeout time.Duration) {
	log.Printf("PingWorker starting. Interval: %s", interval)

	go func() {
		// Run one cycle right away so we have data without waiting a full
		// interval, then settle into the periodic schedule.
		pingAllChannels(s, timeout)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingAllChannels(s, timeout)
			}
		}
	}()
}
