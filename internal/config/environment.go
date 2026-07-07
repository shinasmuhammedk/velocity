package config

import (
	"fmt"
	"os"
	"strings"
)

// Environment represents the application's runtime environment.
type Environment string

const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// String returns the string representation of the environment.
func (e Environment) String() string {
	return string(e)
}

// ConfigFile returns the corresponding configuration file name.
// Example: development -> config.development
func (e Environment) ConfigFile() string {
	return fmt.Sprintf("config.%s", e)
}

// IsDevelopment reports whether the application is running in development mode.
func (e Environment) IsDevelopment() bool {
	return e == Development
}

// IsStaging reports whether the application is running in staging mode.
func (e Environment) IsStaging() bool {
	return e == Staging
}

// IsProduction reports whether the application is running in production mode.
func (e Environment) IsProduction() bool {
	return e == Production
}

// CurrentEnvironment determines the runtime environment.
//
// Priority:
//
// 1. VELOCITY_ENV
// 2. Default -> development
func CurrentEnvironment() Environment {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("VELOCITY_ENV")))

	switch Environment(env) {

	case Production:
		return Production

	case Staging:
		return Staging

	case Development:
		return Development

	default:
		return Development
	}
}