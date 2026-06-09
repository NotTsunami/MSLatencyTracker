// Package handlers contains the Gin HTTP handler functions for the API.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"mslatencytracker/config"
	"mslatencytracker/store"
)

// Handler holds the dependencies the route functions need.
type Handler struct {
	store *store.Store
}

// New builds a Handler.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Register attaches all routes to a router group (e.g. the "/api/v1" group).
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/worlds", h.listWorlds)
	rg.GET("/:world/latency", h.worldLatency)
	rg.GET("/:world/:channel/latency", h.channelLatency)
	rg.GET("/:world/:channel/latency/average", h.channelAverage)
	rg.GET("/:world/:channel/latency/history", h.channelHistory)
}

// listWorlds → GET /api/v1/worlds
func (h *Handler) listWorlds(c *gin.Context) {
	type worldInfo struct {
		Name         string `json:"name"`
		ChannelCount int    `json:"channelCount"`
	}

	worlds := []worldInfo{}
	for _, w := range config.WorldOrder {
		worlds = append(worlds, worldInfo{
			Name:         string(w),
			ChannelCount: config.ChannelCount(w),
		})
	}

	c.JSON(http.StatusOK, worlds)
}

// worldLatency → GET /api/v1/:world/latency
func (h *Handler) worldLatency(c *gin.Context) {
	w, ok := config.TryGetWorld(c.Param("world"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "World not found"})
		return
	}

	channels := h.store.GetAllLatest(w)
	c.JSON(http.StatusOK, gin.H{"world": w, "channels": channels})
}

// channelLatency → GET /api/v1/:world/:channel/latency
func (h *Handler) channelLatency(c *gin.Context) {
	w, ok := config.TryGetWorld(c.Param("world"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "World not found"})
		return
	}

	channel, ok := parseChannel(c, w)
	if !ok {
		return // parseChannel already wrote the 404 response
	}

	reading, ok := h.store.GetLatest(w, channel)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "No data yet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"world":     w,
		"channel":   channel,
		"latencyMs": reading.LatencyMs,
		"timestamp": reading.Timestamp,
	})
}

// channelAverage → GET /api/v1/:world/:channel/latency/average
func (h *Handler) channelAverage(c *gin.Context) {
	w, ok := config.TryGetWorld(c.Param("world"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "World not found"})
		return
	}

	channel, ok := parseChannel(c, w)
	if !ok {
		return
	}

	avg, found, err := h.store.GetAverage(w, channel, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "No data yet"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"world":       w,
		"channel":     channel,
		"averageMs":   avg.AverageMs,
		"sampleCount": avg.SampleCount,
	})
}

// channelHistory → GET /api/v1/:world/:channel/latency/history
func (h *Handler) channelHistory(c *gin.Context) {
	w, ok := config.TryGetWorld(c.Param("world"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "World not found"})
		return
	}

	channel, ok := parseChannel(c, w)
	if !ok {
		return
	}

	points, err := h.store.GetHistory(w, channel, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"world": w, "channel": channel, "dataPoints": points})
}

// parseChannel reads and validates the ":channel" path parameter for a world.
// On any problem it writes a 404 and returns ok=false, so callers can simply
// return. This keeps the validation in one place.
func parseChannel(c *gin.Context, w config.World) (int, bool) {
	channel, err := strconv.Atoi(c.Param("channel"))
	if err != nil || channel < 1 || channel > config.ChannelCount(w) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return 0, false
	}
	return channel, true
}
