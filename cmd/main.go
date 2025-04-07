package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"datadog-proxy/config"
	"datadog-proxy/handler"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func tags(t []string) []string {
	return append([]string{"env:production", "service:datadog-proxy"}, t...)
}

func getLogLevel() slog.Level {
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO", "": // Default to INFO if empty
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo // Default to INFO for unrecognized values
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: getLogLevel()}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger.InfoContext(ctx, "logger and context ready")

	// load environment variables
	c, err := config.New(ctx, logger)
	if err != nil {
		logFatal(logger, "failed to load lambda config", "error", err)
	}
	logger.DebugContext(ctx, "config ready", "config", c)

	if c.TracerEnabled == true {
		tracer.Start(tracer.WithEnv(c.Environment), tracer.WithAgentTimeout(60), tracer.WithSendRetries(5), tracer.WithHTTPClient(&http.Client{
			Timeout: 20 * time.Second, // default 10
			// https://github.com/DataDog/dd-trace-go/blob/e45993802ac1e2c1f0de59e95e18e12de7343266/ddtrace/tracer/transport.go#L45-L52
			Transport: &http.Transport{
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}),
		)
		defer tracer.Stop()
	}

	h := &handler.Handler{
		Config: *c,
		Logger: logger,
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs
		logger.InfoContext(ctx, "received termination signal, exiting")
		os.Exit(0)
	}()

	h.Run(ctx)
}

func logFatal(l *slog.Logger, msg string, args ...any) {
	l.Error(msg, args...)
	os.Exit(1)
}
