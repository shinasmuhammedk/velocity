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
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	GRPC      GRPCConfig      `mapstructure:"grpc"`
	Identity  IdentityConfig  `mapstructure:"identity"`
	Snowflake SnowflakeConfig `mapstructure:"snowflake"`
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
	QueueSize         int  `mapstructure:"queue_size"`
	WorkerCount       int  `mapstructure:"worker_count"`
	SnapshotInterval  int  `mapstructure:"snapshot_interval"`
	PersistenceBuffer int  `mapstructure:"persistence_buffer"`
	RecoveryEnabled   bool `mapstructure:"recovery_enabled"`
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

// MetricsConfig controls the Prometheus scrape endpoint.
//
// The endpoint is served on its own listener (see
// internal/infrastructure/metrics.Server), not on the public API port,
// so Host should normally stay on loopback or a private interface.
//
// Port applies to cmd/api. WorkerPort applies to cmd/worker, which runs
// as a separate process and therefore cannot share a port with the API
// when both are deployed on one host.
type MetricsConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	WorkerPort int    `mapstructure:"worker_port"`
	Path       string `mapstructure:"path"`
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
	Brokers  []string `mapstructure:"brokers"`
	Topic    string   `mapstructure:"topic"`
	DLQTopic string   `mapstructure:"dlq_topic"`
	GroupID  string   `mapstructure:"group_id"`
}

type RateLimitConfig struct {
	Enabled bool `mapstructure:"enabled"`

	SubmitRate  float64 `mapstructure:"submit_rate"`
	SubmitBurst int     `mapstructure:"submit_burst"`

	CancelRate  float64 `mapstructure:"cancel_rate"`
	CancelBurst int     `mapstructure:"cancel_burst"`

	ModifyRate  float64 `mapstructure:"modify_rate"`
	ModifyBurst int     `mapstructure:"modify_burst"`
}

//
// gRPC (Velocity's own server, which the Identity Service calls to
// provision users via VelocityService.CreateUser)
//

type GRPCConfig struct {
	// ListenAddress is where Velocity's own gRPC server binds, e.g. ":50053".
	ListenAddress string `mapstructure:"listen_address"`
}

//
// Identity Service (external, delegated auth - see README "Auth is
// delegated")
//

type IdentityConfig struct {
	// Address is the Identity Service's gRPC address that Velocity
	// dials to validate bearer tokens (AuthService.ValidateToken).
	Address string `mapstructure:"address"`
}

//
// Snowflake ID generation
//
// There are two independent generators in this codebase, seeded
// separately because they produce IDs for different tables (orders vs.
// trades) and there is no reason to force them onto the same node ID:
//   - OrderNodeID seeds container.IDGenerator (internal/app/bootstrap.go),
//     used for order IDs.
//   - TradeNodeID seeds the package-level generator in pkg/idgen, used
//     for trade IDs generated inside the matcher hot path.
//
// Every process that generates IDs concurrently (each cmd/api replica,
// each cmd/matchnode instance) MUST use distinct values here, or IDs can
// collide across instances. There is currently no automatic per-instance
// assignment (e.g. from a StatefulSet pod ordinal) - this must be set
// explicitly per deployment.
type SnowflakeConfig struct {
	OrderNodeID int64 `mapstructure:"order_node_id"`
	TradeNodeID int64 `mapstructure:"trade_node_id"`
}