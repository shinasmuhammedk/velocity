package config

import (
	"fmt"
	"strings"
	"velocity/pkg/errors"
)

// Validate validates the entire application configuration.
func Validate(cfg *Config) error {
	if cfg == nil {
		return errors.ErrConfigMissing
	}

	if err := validateApp(cfg.App); err != nil {
		return err
	}

	if err := validateServer(cfg.Server); err != nil {
		return err
	}

	if err := validateDatabase(cfg.Database); err != nil {
		return err
	}

	if err := validateLogger(cfg.Logger); err != nil {
		return err
	}

	if err := validateEngine(cfg.Engine); err != nil {
		return err
	}

	if err := validateJWT(cfg.JWT); err != nil {
		return err
	}

	return nil
}

// ----------------------------------------------------
// App
// ----------------------------------------------------

func validateApp(cfg AppConfig) error {
	if strings.TrimSpace(cfg.Name) == "" {
		return errors.Wrap(
			errors.CodeConfigMissing,
			"app.name is required",
			nil,
		)
	}

	if strings.TrimSpace(cfg.Environment) == "" {
		return errors.NewConfigMissing("app.environment")
	}

	if strings.TrimSpace(cfg.Version) == "" {
		return errors.NewConfigMissing("app.version")
	}

	return nil
}

// ----------------------------------------------------
// Server
// ----------------------------------------------------

func validateServer(cfg ServerConfig) error {
	if cfg.Host == "" {
		return errors.NewConfigMissing("server.host")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return errors.NewConfigInvalid(
			"server.port",
			"must be between 1 and 65535",
		)
	}

	return nil
}

// ----------------------------------------------------
// Database
// ----------------------------------------------------

func validateDatabase(cfg DatabaseConfig) error {
	if cfg.Host == "" {
		return errors.NewConfigMissing("database.host is required")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return errors.NewConfigMissing("database.port is invalid")
	}

	if cfg.User == "" {
		return errors.NewConfigMissing("database.user is required")
	}

	if cfg.Name == "" {
		return errors.NewConfigMissing("database.name is required")
	}

	return nil
}

// ----------------------------------------------------
// Logger
// ----------------------------------------------------

func validateLogger(cfg LoggerConfig) error {
	switch strings.ToLower(cfg.Level) {

	case "debug",
		"info",
		"warn",
		"error",
		"panic",
		"fatal":
		return nil

	default:
		return fmt.Errorf("invalid logger level: %s", cfg.Level)
	}
}

// ----------------------------------------------------
// Engine
// ----------------------------------------------------

func validateEngine(cfg EngineConfig) error {
	if cfg.QueueSize <= 0 {
		return errors.NewConfigMissing("engine.queue_size must be greater than zero")
	}

	if cfg.WorkerCount <= 0 {
		return errors.NewConfigMissing("engine.worker_count must be greater than zero")
	}

	if cfg.PersistenceBuffer <= 0 {
		return errors.NewConfigMissing("engine.persistence_buffer must be greater than zero")
	}

	return nil
}

// ----------------------------------------------------
// JWT
// ----------------------------------------------------

func validateJWT(cfg JWTConfig) error {
	if strings.TrimSpace(cfg.Secret) == "" {
		return errors.NewConfigMissing("jwt.secret is required")
	}

	if strings.TrimSpace(cfg.Issuer) == "" {
		return errors.NewConfigMissing("jwt.issuer is required")
	}

	return nil
}
