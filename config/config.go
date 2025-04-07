package config

import (
	"context"
	"log/slog"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	DatadogBaseURL string `envconfig:"DATADOG_BASE_URL"`
	TracerEnabled  bool   `envconfig:"DD_ENABLED"`
	Environment    string `envconfig:"ENVIRONMENT"`
}

func New(ctx context.Context, logger *slog.Logger) (*Config, error) {
	c := Config{}
	err := envconfig.Process("", &c)
	if err != nil {
		return nil, err
	}
	logger.DebugContext(ctx, "configuration loaded", "config", c)
	return &c, nil
}
