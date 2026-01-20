// Package main provides the SatGate OSS gateway entrypoint.
//
// SatGate is an Economic Firewall for APIs, providing cryptographic
// capability verification and L402 payment protocol support.
//
// Usage:
//
//	satgate --config gateway.yaml
//	satgate --help
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/satgate-io/satgate/internal/config"
	"github.com/satgate-io/satgate/internal/governance"
	"github.com/satgate-io/satgate/internal/lightning"
	"github.com/satgate-io/satgate/internal/macaroon"
	"github.com/satgate-io/satgate/internal/proxy"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time
	Commit = "unknown"
	// BuildDate is set at build time
	BuildDate = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "gateway.yaml", "Path to configuration file")
	listenAddr := flag.String("listen", "", "Override listen address (e.g., :8080)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	jsonLogs := flag.Bool("json-logs", false, "Output logs in JSON format")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// Version flag
	if *showVersion {
		fmt.Printf("SatGate OSS %s\n", Version)
		fmt.Printf("  Commit:     %s\n", Commit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Policies:   public, capability, l402\n")
		os.Exit(0)
	}

	// Setup logging
	setupLogging(*logLevel, *jsonLogs)

	log.Info().
		Str("version", Version).
		Str("commit", Commit).
		Msg("Starting SatGate OSS Gateway")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", *configPath).Msg("Failed to load configuration")
	}

	// Override listen address if provided
	if *listenAddr != "" {
		cfg.Server.Listen = *listenAddr
	}

	// Get listen address
	listen := cfg.Server.Listen
	if listen == "" {
		listen = ":8080"
	}

	// Initialize macaroon service
	rootKey := cfg.Admin.CapabilityRootKey
	if rootKey == "" {
		rootKey = os.Getenv("CAPABILITY_ROOT_KEY")
	}
	if rootKey == "" {
		log.Fatal().Msg("CAPABILITY_ROOT_KEY is required (set in config or environment)")
	}

	macaroonSvc, err := macaroon.NewService(rootKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize macaroon service")
	}
	log.Info().Msg("Macaroon service initialized")

	// Initialize governance service
	governanceSvc := governance.NewService(governance.NewMemoryStore())
	log.Info().Msg("Governance service initialized (in-memory)")

	// Initialize lightning provider (optional)
	var lightningSvc lightning.Provider
	if cfg.Lightning.Provider != "" && cfg.Lightning.Provider != "disabled" {
		lightningSvc, err = lightning.NewProvider(cfg.Lightning.Provider, cfg.Lightning.Config)
		if err != nil {
			log.Warn().Err(err).Str("provider", cfg.Lightning.Provider).
				Msg("Failed to initialize lightning provider - L402 routes will be disabled")
		} else {
			log.Info().Str("provider", cfg.Lightning.Provider).Msg("Lightning provider initialized")
		}
	}

	// Create gateway
	gw, err := proxy.New(proxy.Options{
		Config:     cfg,
		Macaroon:   macaroonSvc,
		Governance: governanceSvc,
		Lightning:  lightningSvc,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create gateway")
	}

	// Log routes
	log.Info().Int("routes", len(cfg.Routes)).Msg("Routes configured")
	for _, route := range cfg.Routes {
		policy := "capability"
		if route.Policy.Kind != "" {
			policy = route.Policy.Kind
		}
		path := route.Match.PathPrefix
		if route.Match.PathExact != "" {
			path = route.Match.PathExact
		}
		log.Debug().Str("name", route.Name).Str("path", path).Str("policy", policy).Msg("Route")
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         listen,
		Handler:      gw,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("Shutting down gateway...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error during shutdown")
		}
		close(done)
	}()

	// Start server
	log.Info().Str("listen", listen).Msg("Gateway listening")
	log.Info().Msg("Supported policies: public, capability, l402")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server error")
	}

	<-done
	log.Info().Msg("Gateway stopped")
}

func setupLogging(level string, jsonFormat bool) {
	// Set log level
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Set output format
	if jsonFormat {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}
