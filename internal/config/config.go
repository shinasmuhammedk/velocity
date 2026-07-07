package config

import "time"

// Config is the root configuration object for the entire application.
// Every subsystem receives only the configuration it requires.
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Logger    LoggerConfig    `mapstructure:"logger"`
	Engine    EngineConfig    `mapstructure:"engine"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	Tracing   TracingConfig   `mapstructure:"tracing"`
}

//
// Application
//

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Version     string `mapstructure:"version"`
}

//
// HTTP Server
//

type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

//
// PostgreSQL
//

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

//
// Logger
//

type LoggerConfig struct {
	Level       string `mapstructure:"level"`
	Encoding    string `mapstructure:"encoding"`
	Development bool   `mapstructure:"development"`
}

//
// Matching Engine
//

type EngineConfig struct {
	QueueSize          int  `mapstructure:"queue_size"`
	WorkerCount        int  `mapstructure:"worker_count"`
	SnapshotInterval   int  `mapstructure:"snapshot_interval"`
	PersistenceBuffer  int  `mapstructure:"persistence_buffer"`
	RecoveryEnabled    bool `mapstructure:"recovery_enabled"`
}

//
// Authentication
//

type JWTConfig struct {
	Secret string        `mapstructure:"secret"`
	Issuer string        `mapstructure:"issuer"`
	Expiry time.Duration `mapstructure:"expiry"`
}

//
// WebSocket
//

type WebSocketConfig struct {
	ReadBufferSize  int `mapstructure:"read_buffer_size"`
	WriteBufferSize int `mapstructure:"write_buffer_size"`
	MaxConnections  int `mapstructure:"max_connections"`
}

//
// Metrics
//

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

//
// Redis
//

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
}

//
// Kafka / NATS
//

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
	GroupID string   `mapstructure:"group_id"`
}

//
// Distributed Tracing
//

type TracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Exporter string `mapstructure:"exporter"`
	Endpoint string `mapstructure:"endpoint"`
}