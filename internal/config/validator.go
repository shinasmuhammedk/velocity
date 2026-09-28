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

	if err := validateKafka(cfg.Kafka); err != nil {
		return err
	}

	if err := validateMetrics(cfg.Metrics, cfg.Server); err != nil {
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

// ----------------------------------------------------
// Kafka
// ----------------------------------------------------

func validateKafka(cfg KafkaConfig) error {
	if len(cfg.Brokers) == 0 {
		return errors.NewConfigMissing("kafka.brokers is required")
	}

	for _, broker := range cfg.Brokers {
		if strings.TrimSpace(broker) == "" {
			return errors.NewConfigMissing("kafka.brokers must not contain empty entries")
		}
	}

	if strings.TrimSpace(cfg.Topic) == "" {
		return errors.NewConfigMissing("kafka.topic is required")
	}

	if strings.TrimSpace(cfg.DLQTopic) == "" {
		return errors.NewConfigMissing("kafka.dlq_topic is required")
	}

	if cfg.Topic == cfg.DLQTopic {
		return errors.NewConfigMissing("kafka.topic and kafka.dlq_topic must be different")
	}

	if strings.TrimSpace(cfg.GroupID) == "" {
		return errors.NewConfigMissing("kafka.group_id is required")
	}

	return nil
}

// ----------------------------------------------------
// Metrics
// ----------------------------------------------------

// validateMetrics checks the Prometheus listener settings.
//
// Nothing is validated when metrics are disabled, since the remaining
// fields are then unused. When enabled, the API metrics port must not
// collide with the API's own HTTP port or with the worker's metrics
// port - both are silent misconfigurations that would otherwise only
// surface as a bind failure at startup, or worse, as a metrics endpoint
// accidentally sharing the public listener.
func validateMetrics(cfg MetricsConfig, server ServerConfig) error {
	if !cfg.Enabled {
		return nil
	}

	if strings.TrimSpace(cfg.Host) == "" {
		return errors.NewConfigMissing("metrics.host")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return errors.NewConfigInvalid(
			"metrics.port",
			"must be between 1 and 65535",
		)
	}

	if cfg.WorkerPort < 1 || cfg.WorkerPort > 65535 {
		return errors.NewConfigInvalid(
			"metrics.worker_port",
			"must be between 1 and 65535",
		)
	}

	if cfg.Port == cfg.WorkerPort {
		return errors.NewConfigInvalid(
			"metrics.worker_port",
			"must differ from metrics.port",
		)
	}

	if cfg.Port == server.Port {
		return errors.NewConfigInvalid(
			"metrics.port",
			"must differ from server.port - metrics are served on a separate, non-public listener",
		)
	}

	if !strings.HasPrefix(strings.TrimSpace(cfg.Path), "/") {
		return errors.NewConfigInvalid(
			"metrics.path",
			"must start with /",
		)
	}

	return nil
}