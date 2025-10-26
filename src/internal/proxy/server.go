package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"go-oauth2-proxy/src/internal/config"
	"go-oauth2-proxy/src/internal/logger"
	"go-oauth2-proxy/src/internal/token"
)

// Server represents the proxy server
type Server struct {
	config       *config.Config
	tokenManager *token.Manager
	httpServer   *http.Server
	upstreamMap  map[string]*config.UpstreamConfig
}

// NewServer creates a new proxy server
func NewServer(cfg *config.Config) (*Server, error) {
	// Create token manager
	tm := token.NewManager(
		context.Background(),
		"", // Will use GOOGLE_APPLICATION_CREDENTIALS env var
		cfg.Token.RefreshBeforeExpiry,
	)

	// Build upstream map
	upstreamMap := make(map[string]*config.UpstreamConfig)
	for i := range cfg.Upstreams {
		upstreamMap[cfg.Upstreams[i].Name] = &cfg.Upstreams[i]
	}

	srv := &Server{
		config:       cfg,
		tokenManager: tm,
		upstreamMap:  upstreamMap,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.handleHealth)
	mux.HandleFunc("/readyz", srv.handleReady)
	mux.HandleFunc("/metrics", srv.handleMetrics)
	mux.HandleFunc("/token-info", srv.handleTokenInfo)
	mux.HandleFunc("/", srv.handleProxy)

	srv.httpServer = &http.Server{
		Addr:         cfg.Server.GetAddress(),
		Handler:      srv.loggingMiddleware(mux),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	return srv, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	logger.Info("Starting HTTP server",
		"address", s.httpServer.Addr,
		"upstreams", len(s.config.Upstreams))

	for _, upstream := range s.config.Upstreams {
		logger.Info("Configured upstream",
			"name", upstream.Name,
			"url", upstream.URL,
			"audience", upstream.Audience)
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		logger.Info("Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"user_agent", r.Header.Get("User-Agent"))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReady handles readiness check requests
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}

// handleMetrics returns server metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats := s.tokenManager.GetStats()

	metrics := map[string]interface{}{
		"tokens_cached":    stats.TotalCached,
		"tokens_refreshed": stats.TotalRefreshed,
		"tokens_rejected":  stats.TotalRejected,
		"tokens_errors":    stats.TotalErrors,
		"upstreams_count":  len(s.config.Upstreams),
	}

	if stats.TotalCached > 0 {
		metrics["oldest_token_age"] = time.Since(stats.OldestToken).String()
		metrics["newest_token_age"] = time.Since(stats.NewestToken).String()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleTokenInfo returns detailed token information
func (s *Server) handleTokenInfo(w http.ResponseWriter, r *http.Request) {
	allMetadata := s.tokenManager.GetAllMetadata()

	response := make(map[string]interface{})
	response["total_tokens"] = len(allMetadata)
	response["upstreams_configured"] = len(s.config.Upstreams)

	tokens := make([]map[string]interface{}, 0)
	for audience, meta := range allMetadata {
		tokenInfo := map[string]interface{}{
			"audience":       audience,
			"state":          meta.State,
			"issued_at":      meta.IssuedAt.Format(time.RFC3339),
			"expires_at":     meta.ExpiresAt.Format(time.RFC3339),
			"expires_in":     time.Until(meta.ExpiresAt).String(),
			"last_used":      meta.LastUsed.Format(time.RFC3339),
			"refresh_count":  meta.RefreshCount,
			"rejected_count": meta.RejectedCount,
			"error_count":    meta.ErrorCount,
		}
		if meta.LastError != "" {
			tokenInfo["last_error"] = meta.LastError
		}
		tokens = append(tokens, tokenInfo)
	}
	response["tokens"] = tokens

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleProxy handles proxy requests
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Determine upstream
	upstream := s.determineUpstream(r)
	if upstream == nil {
		logger.Warn("No upstream found", "path", r.URL.Path)
		http.Error(w, "No upstream configured for this request", http.StatusNotFound)
		return
	}

	logger.Debug("Proxying request",
		"method", r.Method,
		"path", r.URL.Path,
		"upstream", upstream.Name,
		"target", upstream.URL)

	// Get token for upstream
	token, err := s.tokenManager.GetToken(upstream.Audience)
	if err != nil {
		logger.Error("Failed to get token",
			"upstream", upstream.Name,
			"audience", upstream.Audience,
			"error", err)
		http.Error(w, fmt.Sprintf("Authentication error: %v", err), http.StatusInternalServerError)
		return
	}

	// Parse upstream URL
	targetURL, err := url.Parse(upstream.URL)
	if err != nil {
		logger.Error("Invalid upstream URL",
			"upstream", upstream.Name,
			"url", upstream.URL,
			"error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = singleJoiningSlash(targetURL.Path, req.URL.Path)
			req.Host = targetURL.Host

			// Add authorization header
			req.Header.Set("Authorization", "Bearer "+token)

			// Set forwarded headers
			if clientIP := req.Header.Get("X-Forwarded-For"); clientIP == "" {
				req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			}
			req.Header.Set("X-Forwarded-Proto", "https")

			// Remove hop-by-hop headers
			for _, h := range hopHeaders {
				req.Header.Del(h)
			}

			logger.Debug("Upstream request",
				"method", req.Method,
				"url", req.URL.String(),
				"upstream", upstream.Name)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("Proxy error",
				"upstream", upstream.Name,
				"error", err,
				"duration_ms", time.Since(startTime).Milliseconds())
			http.Error(w, fmt.Sprintf("Bad Gateway: %v", err), http.StatusBadGateway)
		},
		ModifyResponse: func(resp *http.Response) error {
			// Check for authentication errors
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				logger.Warn("Upstream rejected token",
					"upstream", upstream.Name,
					"status", resp.StatusCode,
					"duration_ms", time.Since(startTime).Milliseconds())
				s.tokenManager.MarkRejected(upstream.Audience)
			}

			logger.Debug("Upstream response",
				"upstream", upstream.Name,
				"status", resp.StatusCode,
				"duration_ms", time.Since(startTime).Milliseconds())

			return nil
		},
	}

	proxy.ServeHTTP(w, r)
}

// determineUpstream selects the appropriate upstream for the request
func (s *Server) determineUpstream(r *http.Request) *config.UpstreamConfig {
	// Check X-Target-Upstream header
	targetName := r.Header.Get("X-Target-Upstream")
	if targetName != "" {
		if upstream, exists := s.upstreamMap[targetName]; exists {
			return upstream
		}
		logger.Warn("Upstream not found", "name", targetName)
	}

	// Default to first upstream
	if len(s.config.Upstreams) > 0 {
		return &s.config.Upstreams[0]
	}

	return nil
}

// Hop-by-hop headers to remove
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// singleJoiningSlash joins two URL paths
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
