// Command satgate-mcp runs the SatGate MCP proxy gateway.
//
// It proxies MCP protocol traffic between an agent (Claude Code, Agent Zero, etc.)
// and upstream MCP servers, enforcing per-tool budgets.
//
// Usage:
//
//	satgate-mcp --config mcp-proxy.yaml
//	satgate-mcp --help
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/mcpserver"
)

// The gateway's governance service is the MCP runtime's lineage ban check.
var _ mcpserver.LineageRevocationChecker = (*governance.Service)(nil)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	configPath := flag.String("config", "satgate-mcp.yaml", "Path to MCP proxy configuration")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("SatGate MCP Proxy %s\n", Version)
		fmt.Printf("  Commit:     %s\n", Commit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// Load config
	cfg, err := mcpserver.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	setupLogging(cfg.Logging.Level, cfg.Logging.JSON)

	log.Info().
		Str("version", Version).
		Str("transport", cfg.Server.Transport).
		Str("enforcement", cfg.Enforcement.Mode).
		Int64("budget", cfg.Budget.Limit).
		Msg("Starting SatGate MCP Proxy")

	// Create proxy
	proxy, err := mcpserver.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create MCP proxy")
	}

	// Token bans: every macaroon a ban covers (the banned token and every
	// token delegated from it) is refused on every transport.
	bans, err := newRevocationService(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open the shared ban store")
	}
	defer bans.Close()
	proxy.SetLineageRevocationChecker(bans)

	// Context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Info().Msg("Shutting down MCP proxy...")
		cancel()
	}()

	// Run based on transport mode
	switch cfg.Server.Transport {
	case "sse", "http":
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		if cfg.Server.Port == 0 {
			addr = ":9100"
		}
		sseServer := mcpserver.NewSSEServer(proxy, addr)
		if err := sseServer.ListenAndServe(ctx); err != nil {
			log.Error().Err(err).Msg("SSE server error")
			os.Exit(1)
		}

	default: // stdio
		clientTransport := mcpserver.NewStdioTransport(os.Stdin, os.Stdout, nil)
		if err := proxy.Run(ctx, clientTransport); err != nil {
			log.Error().Err(err).Msg("MCP proxy error")
			os.Exit(1)
		}
	}

	log.Info().Msg("MCP proxy stopped")
}

// newRevocationService returns the ban list the proxy checks: the shared
// store (the Redis a SatGate gateway writes its bans to) when
// revocation.redis.addr is set, so a ban written through any gateway on that
// Redis is honoured here within governance.BanCacheTTL; otherwise this
// process's memory, so bans apply to this process only. A configured store
// that cannot be reached is an error, never a silent fallback to memory.
func newRevocationService(cfg *mcpserver.Config) (*governance.Service, error) {
	rc := cfg.Revocation.Redis
	if rc.Addr == "" {
		log.Info().Msg("Token bans kept in memory: they apply to this process only")
		return governance.NewService(governance.NewMemoryStore()), nil
	}
	store, err := governance.NewRedisStore(&governance.RedisConfig{Addr: rc.Addr, Password: rc.Password, DB: rc.DB})
	if err != nil {
		return nil, err
	}
	log.Info().Dur("ban_cache_ttl", governance.BanCacheTTL).Dur("read_timeout", governance.BanStoreReadTimeout).
		Msg("Token bans read from the shared store")
	return governance.NewService(store), nil
}

func setupLogging(level string, jsonFormat bool) {
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if jsonFormat {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		// MCP uses stdout for protocol — logs MUST go to stderr
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}
