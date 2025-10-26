package token

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"

	"github.com/yourorg/token-gateway/internal/logger"
)

// TokenState represents the current state of a token
type TokenState string

const (
	StateNew       TokenState = "NEW"        // Token not yet created
	StateCached    TokenState = "CACHED"     // Token cached and valid
	StateRefreshed TokenState = "REFRESHED"  // Token was refreshed
	StateExpiring  TokenState = "EXPIRING"   // Token expiring soon
	StateExpired   TokenState = "EXPIRED"    // Token expired
	StateRejected  TokenState = "REJECTED"   // Token rejected by upstream
	StateError     TokenState = "ERROR"      // Error getting token
)

// TokenMetadata holds metadata about a cached token
type TokenMetadata struct {
	Audience      string
	State         TokenState
	Token         string
	IssuedAt      time.Time
	ExpiresAt     time.Time
	LastUsed      time.Time
	RefreshCount  int
	RejectedCount int
	ErrorCount    int
	LastError     string
}

// TokenEntry represents a cached token with its source
type TokenEntry struct {
	tokenSource oauth2.TokenSource
	metadata    *TokenMetadata
	mu          sync.RWMutex
}

// Manager handles token creation, caching, and refresh
type Manager struct {
	cache              map[string]*TokenEntry
	cacheMu            sync.RWMutex
	ctx                context.Context
	credsFile          string
	refreshBeforeExpiry time.Duration
}

// NewManager creates a new token manager
func NewManager(ctx context.Context, credsFile string, refreshBeforeMinutes int) *Manager {
	return &Manager{
		cache:              make(map[string]*TokenEntry),
		ctx:                ctx,
		credsFile:          credsFile,
		refreshBeforeExpiry: time.Duration(refreshBeforeMinutes) * time.Minute,
	}
}

// GetToken returns a valid token for the given audience
func (m *Manager) GetToken(audience string) (string, error) {
	m.cacheMu.Lock()
	entry, exists := m.cache[audience]
	if !exists {
		// Create new entry
		entry = &TokenEntry{
			metadata: &TokenMetadata{
				Audience:  audience,
				State:     StateNew,
				IssuedAt:  time.Now(),
			},
		}
		m.cache[audience] = entry
	}
	m.cacheMu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Check if we need to refresh
	if m.shouldRefresh(entry) {
		if err := m.refreshToken(entry, audience); err != nil {
			entry.metadata.State = StateError
			entry.metadata.ErrorCount++
			entry.metadata.LastError = err.Error()
			logger.Error("Failed to get/refresh token",
				"audience", audience,
				"error", err,
				"error_count", entry.metadata.ErrorCount)
			return "", err
		}
	}

	// Update last used
	entry.metadata.LastUsed = time.Now()

	logger.Debug("Token retrieved",
		"audience", audience,
		"state", entry.metadata.State,
		"expires_in", time.Until(entry.metadata.ExpiresAt).String(),
		"refresh_count", entry.metadata.RefreshCount)

	return entry.metadata.Token, nil
}

// shouldRefresh determines if a token needs to be refreshed
func (m *Manager) shouldRefresh(entry *TokenEntry) bool {
	meta := entry.metadata

	// New token - needs creation
	if meta.State == StateNew {
		return true
	}

	// No token source - needs creation
	if entry.tokenSource == nil {
		return true
	}

	// Token expired
	if time.Now().After(meta.ExpiresAt) {
		meta.State = StateExpired
		return true
	}

	// Token expiring soon
	if time.Now().Add(m.refreshBeforeExpiry).After(meta.ExpiresAt) {
		if meta.State != StateExpiring {
			logger.Info("Token expiring soon, will refresh",
				"audience", meta.Audience,
				"expires_in", time.Until(meta.ExpiresAt).String())
			meta.State = StateExpiring
		}
		return true
	}

	return false
}

// refreshToken creates or refreshes a token
func (m *Manager) refreshToken(entry *TokenEntry, audience string) error {
	meta := entry.metadata
	startTime := time.Now()

	logger.Info("Refreshing token",
		"audience", audience,
		"state", meta.State,
		"refresh_count", meta.RefreshCount)

	// Create token source if needed
	if entry.tokenSource == nil {
		ts, err := idtoken.NewTokenSource(m.ctx, audience,
			idtoken.WithCredentialsFile(m.credsFile))
		if err != nil {
			return fmt.Errorf("failed to create token source: %w", err)
		}
		entry.tokenSource = ts
		logger.Debug("Token source created", "audience", audience)
	}

	// Get token
	token, err := entry.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Update metadata
	meta.Token = token.AccessToken
	meta.ExpiresAt = token.Expiry
	meta.RefreshCount++
	meta.LastError = ""

	if meta.State == StateNew {
		meta.State = StateCached
		logger.Info("New token created",
			"audience", audience,
			"expires_at", token.Expiry.Format(time.RFC3339),
			"valid_for", time.Until(token.Expiry).String(),
			"duration", time.Since(startTime).String())
	} else {
		meta.State = StateRefreshed
		logger.Info("Token refreshed",
			"audience", audience,
			"expires_at", token.Expiry.Format(time.RFC3339),
			"valid_for", time.Until(token.Expiry).String(),
			"refresh_count", meta.RefreshCount,
			"duration", time.Since(startTime).String())
	}

	return nil
}

// MarkRejected marks a token as rejected (e.g., 401/403 from upstream)
func (m *Manager) MarkRejected(audience string) {
	m.cacheMu.RLock()
	entry, exists := m.cache[audience]
	m.cacheMu.RUnlock()

	if !exists {
		return
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.metadata.State = StateRejected
	entry.metadata.RejectedCount++

	logger.Warn("Token rejected by upstream",
		"audience", audience,
		"rejected_count", entry.metadata.RejectedCount)

	// Force refresh on next request
	entry.tokenSource = nil
}

// GetMetadata returns metadata for a specific audience
func (m *Manager) GetMetadata(audience string) *TokenMetadata {
	m.cacheMu.RLock()
	entry, exists := m.cache[audience]
	m.cacheMu.RUnlock()

	if !exists {
		return nil
	}

	entry.mu.RLock()
	defer entry.mu.RUnlock()

	// Create a copy to avoid race conditions
	meta := *entry.metadata
	return &meta
}

// GetAllMetadata returns metadata for all cached tokens
func (m *Manager) GetAllMetadata() map[string]*TokenMetadata {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	result := make(map[string]*TokenMetadata)
	for audience, entry := range m.cache {
		entry.mu.RLock()
		meta := *entry.metadata
		entry.mu.RUnlock()
		result[audience] = &meta
	}

	return result
}

// Stats returns aggregate statistics
type Stats struct {
	TotalCached     int
	TotalRefreshed  int
	TotalRejected   int
	TotalErrors     int
	OldestToken     time.Time
	NewestToken     time.Time
}

func (m *Manager) GetStats() Stats {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	stats := Stats{}
	first := true

	for _, entry := range m.cache {
		entry.mu.RLock()
		meta := entry.metadata

		stats.TotalCached++
		stats.TotalRefreshed += meta.RefreshCount
		stats.TotalRejected += meta.RejectedCount
		stats.TotalErrors += meta.ErrorCount

		if first {
			stats.OldestToken = meta.IssuedAt
			stats.NewestToken = meta.IssuedAt
			first = false
		} else {
			if meta.IssuedAt.Before(stats.OldestToken) {
				stats.OldestToken = meta.IssuedAt
			}
			if meta.IssuedAt.After(stats.NewestToken) {
				stats.NewestToken = meta.IssuedAt
			}
		}

		entry.mu.RUnlock()
	}

	return stats
}
