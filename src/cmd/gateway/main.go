package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-oauth2-proxy/src/internal/config"
	"go-oauth2-proxy/src/internal/logger"
	"go-oauth2-proxy/src/internal/proxy"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	credsPath := flag.String("credentials", "", "Path to GCP service account JSON file (or set GOOGLE_APPLICATION_CREDENTIALS)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Initialize logger
	logger.Init(*logLevel)
	logger.Info("Starting Token Gateway")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
	}
	logger.Info("Configuration loaded", "upstreams", len(cfg.Upstreams))

	// Set credentials path
	if *credsPath != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", *credsPath)
	}

	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		logger.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
	}
	logger.Info("Using credentials file", "path", credsFile)

	// Create and start proxy server
	srv, err := proxy.NewServer(cfg)
	if err != nil {
		logger.Fatal("Failed to create proxy server", "error", err)
	}

	// Start server in a goroutine
	go func() {
		addr := cfg.Server.GetAddress()
		logger.Info("Server starting", "address", addr)
		if err := srv.Start(); err != nil {
			logger.Fatal("Server failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	if err := srv.Shutdown(); err != nil {
		logger.Error("Server shutdown failed", "error", err)
	}
	logger.Info("Server stopped")
}
