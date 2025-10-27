package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
	logger.Debug("NewServer called")

	// Get credentials file from environment
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		logger.Error("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
	}

	logger.Info("Using credentials file for token manager", "path", credsFile)
	logger.Debug("Token manager config",
		"creds_file", credsFile,
		"refresh_before_expiry", cfg.Token.RefreshBeforeExpiry)

	// Create token manager WITH credentials file
	tm := token.NewManager(
		context.Background(),
		credsFile, // Pass the actual credentials file path
		cfg.Token.RefreshBeforeExpiry,
	)

	logger.Debug("Token manager created successfully")

	// Build upstream map
	upstreamMap := make(map[string]*config.UpstreamConfig)
	for i := range cfg.Upstreams {
		upstreamMap[cfg.Upstreams[i].Name] = &cfg.Upstreams[i]
		logger.Debug("Added upstream to map",
			"name", cfg.Upstreams[i].Name,
			"url", cfg.Upstreams[i].URL,
			"audience", cfg.Upstreams[i].Audience,
			"host", cfg.Upstreams[i].Host)
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

	logger.Debug("HTTP server configured",
		"address", srv.httpServer.Addr,
		"read_timeout", cfg.Server.ReadTimeout,
		"write_timeout", cfg.Server.WriteTimeout,
		"idle_timeout", cfg.Server.IdleTimeout)

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
			"audience", upstream.Audience,
			"host", upstream.Host)
	}

	logger.Debug("Calling ListenAndServe")
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	logger.Info("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		logger.Error("Server shutdown error", "error", err)
	} else {
		logger.Info("Server shutdown completed")
	}
	return err
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log incoming request
		logger.Debug("Incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.Header.Get("User-Agent"),
			"content_length", r.ContentLength,
			"host", r.Host)

		// Log all headers in debug mode
		logger.Debug("Request headers")
		for name, values := range r.Header {
			for _, value := range values {
				logger.Debug("Header", "name", name, "value", value)
			}
		}

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		logger.Info("Request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"user_agent", r.Header.Get("User-Agent"))

		logger.Debug("Response details",
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"bytes_written", wrapped.bytesWritten)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Health check request")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReady handles readiness check requests
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Readiness check request")
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("READY"))
}

// handleMetrics returns server metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Metrics request")
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
	logger.Debug("Metrics response sent", "stats", metrics)
}

// handleTokenInfo returns detailed token information
func (s *Server) handleTokenInfo(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Token info request")
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
	logger.Debug("Token info response sent", "token_count", len(tokens))
}

// handleProxy handles proxy requests
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	logger.Debug("handleProxy called",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery)

	// Check if path is allowed (if filtering is enabled)
	if !s.isPathAllowed(r.URL.Path) {
		logger.Warn("Path not allowed", "path", r.URL.Path, "remote_addr", r.RemoteAddr)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Determine upstream
	upstream := s.determineUpstream(r)
	if upstream == nil {
		logger.Warn("No upstream found", "path", r.URL.Path)
		http.Error(w, "No upstream configured for this request", http.StatusNotFound)
		return
	}

	logger.Debug("Upstream determined",
		"name", upstream.Name,
		"url", upstream.URL,
		"audience", upstream.Audience,
		"host", upstream.Host)

	logger.Debug("Proxying request",
		"method", r.Method,
		"path", r.URL.Path,
		"upstream", upstream.Name,
		"target", upstream.URL)

	// Get token for upstream
	logger.Debug("Requesting token", "audience", upstream.Audience)
	token, err := s.tokenManager.GetToken(upstream.Audience)
	if err != nil {
		logger.Error("Failed to get token",
			"upstream", upstream.Name,
			"audience", upstream.Audience,
			"error", err)
		http.Error(w, fmt.Sprintf("Authentication error: %v", err), http.StatusInternalServerError)
		return
	}
	logger.Debug("Token obtained successfully", "token_length", len(token))

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

	logger.Debug("Target URL parsed",
		"scheme", targetURL.Scheme,
		"host", targetURL.Host,
		"path", targetURL.Path)

	// Create reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			logger.Debug("Director function called")

			originalURL := req.URL.String()
			originalHost := req.Host

			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = singleJoiningSlash(targetURL.Path, req.URL.Path)

			// Set Host header
			if upstream.Host != "" {
				req.Host = upstream.Host
				logger.Debug("Setting custom Host header",
					"host", upstream.Host,
					"original_host", originalHost)
			} else {
				req.Host = targetURL.Host
				logger.Debug("Using target Host header",
					"host", targetURL.Host,
					"original_host", originalHost)
			}

			// Add authorization header
			req.Header.Set("Authorization", "Bearer "+token)
			logger.Debug("Authorization header set", "token_length", len(token))

			// Set forwarded headers
			if clientIP := req.Header.Get("X-Forwarded-For"); clientIP == "" {
				req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			}
			req.Header.Set("X-Forwarded-Proto", "https")

			// Remove hop-by-hop headers
			for _, h := range hopHeaders {
				req.Header.Del(h)
			}

			logger.Debug("Upstream request prepared",
				"method", req.Method,
				"url", req.URL.String(),
				"host", req.Host,
				"original_url", originalURL,
				"upstream", upstream.Name)

			// Log all outgoing headers
			logger.Debug("Outgoing request headers")
			for name, values := range req.Header {
				for _, value := range values {
					// Redact authorization header value
					if name == "Authorization" {
						logger.Debug("Header", "name", name, "value", "Bearer [REDACTED]")
					} else {
						logger.Debug("Header", "name", name, "value", value)
					}
				}
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("Proxy error",
				"upstream", upstream.Name,
				"url", r.URL.String(),
				"error", err,
				"error_type", fmt.Sprintf("%T", err),
				"duration_ms", time.Since(startTime).Milliseconds())
			http.Error(w, fmt.Sprintf("Bad Gateway: %v", err), http.StatusBadGateway)
		},
		ModifyResponse: func(resp *http.Response) error {
			logger.Debug("Response received from upstream",
				"upstream", upstream.Name,
				"status", resp.StatusCode,
				"status_text", resp.Status,
				"content_length", resp.ContentLength,
				"duration_ms", time.Since(startTime).Milliseconds())

			// Log response headers
			logger.Debug("Upstream response headers")
			for name, values := range resp.Header {
				for _, value := range values {
					logger.Debug("Header", "name", name, "value", value)
				}
			}

			// Check for authentication errors
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				logger.Warn("Upstream rejected token",
					"upstream", upstream.Name,
					"status", resp.StatusCode,
					"duration_ms", time.Since(startTime).Milliseconds())

				// Read and log error body
				if resp.Body != nil {
					bodyBytes, err := io.ReadAll(resp.Body)
					if err == nil {
						logger.Debug("Error response body", "body", string(bodyBytes))
						// Restore body for client
						resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
					}
				}

				s.tokenManager.MarkRejected(upstream.Audience)
			}

			logger.Debug("Upstream response processed",
				"upstream", upstream.Name,
				"status", resp.StatusCode,
				"duration_ms", time.Since(startTime).Milliseconds())

			return nil
		},
	}

	logger.Debug("Starting reverse proxy")
	proxy.ServeHTTP(w, r)
	logger.Debug("Reverse proxy completed", "duration_ms", time.Since(startTime).Milliseconds())
}

// determineUpstream selects the appropriate upstream for the request
func (s *Server) determineUpstream(r *http.Request) *config.UpstreamConfig {
	logger.Debug("Determining upstream")

	// Check X-Target-Upstream header
	targetName := r.Header.Get("X-Target-Upstream")
	if targetName != "" {
		logger.Debug("X-Target-Upstream header found", "name", targetName)
		if upstream, exists := s.upstreamMap[targetName]; exists {
			logger.Debug("Upstream found by header", "name", targetName)
			return upstream
		}
		logger.Warn("Upstream not found by header", "name", targetName)
	}

	// Default to first upstream
	if len(s.config.Upstreams) > 0 {
		logger.Debug("Using default (first) upstream", "name", s.config.Upstreams[0].Name)
		return &s.config.Upstreams[0]
	}

	logger.Warn("No upstreams configured")
	return nil
}

// isPathAllowed checks if the request path is allowed based on configured patterns
func (s *Server) isPathAllowed(path string) bool {
	// If no allowed paths configured, allow all
	if len(s.config.Server.AllowedPaths) == 0 {
		logger.Debug("No path filtering configured, allowing all paths")
		return true
	}

	// Check each allowed pattern
	for _, pattern := range s.config.Server.AllowedPaths {
		if matchPath(pattern, path) {
			logger.Debug("Path allowed", "path", path, "pattern", pattern)
			return true
		}
	}

	logger.Debug("Path not allowed", "path", path)
	return false
}

// matchPath checks if a path matches a pattern
// Supports exact matches and wildcard patterns (e.g., /apps/*)
func matchPath(pattern, path string) bool {
	// Exact match
	if pattern == path {
		return true
	}

	// Wildcard pattern (e.g., /apps/*)
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}

	// Wildcard pattern with ** (e.g., /apps/**)
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(path, prefix+"/") || path == prefix
	}

	return false
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
