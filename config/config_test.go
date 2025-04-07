package config

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

type testLogger struct{}

func (l testLogger) DebugContext(ctx context.Context, msg string, args ...any) {}

func TestNew(t *testing.T) {
	// Set up environment variable
	expectedURL := "https://example.com"
	os.Setenv("DATADOG_BASE_URL", expectedURL)
	defer os.Unsetenv("DATADOG_BASE_URL")

	ctx := context.Background()

	// Using a no-op logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := New(ctx, logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DatadogBaseURL != expectedURL {
		t.Errorf("expected DatadogBaseURL %q, got %q", expectedURL, cfg.DatadogBaseURL)
	}
}
