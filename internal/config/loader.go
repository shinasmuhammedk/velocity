package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load reads the application's configuration from disk,
// applies environment variable overrides,
// validates the final configuration,
// and returns a populated Config.
func Load() (*Config, error) {
	v := viper.New()

	// Where configuration files are located
	v.AddConfigPath("./configs")
	v.AddConfigPath("../configs")
	v.AddConfigPath("../../configs")

	// Configuration file
    env := CurrentEnvironment()
	v.SetConfigName(env.ConfigFile())
	v.SetConfigType("yaml")

	// Environment variables
	v.SetEnvPrefix("VELOCITY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read YAML
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	cfg := new(Config)

	// Convert YAML -> Config struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate configuration
	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
