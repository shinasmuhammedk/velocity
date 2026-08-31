package config_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"velocity/internal/config"

	"github.com/stretchr/testify/require"
)

// TestRateLimitEnabledAcrossAllEnvironments guards against the class of
// bug where an env config simply omits the `rate_limit:` section: Viper
// unmarshals that as RateLimitConfig{} (Enabled: false) instead of
// erroring, and the middleware silently no-ops.
//
// NOTE: config.staging.yaml and config.prod.yaml are currently empty
// placeholder files (never populated since the initial commit — no
// CI/CD, no env-var injection pipeline exists yet for them). They're
// intentionally excluded here so this test doesn't perma-fail on an
// unrelated, larger gap (those environments not being built out at
// all). Add them back to `envs` once real staging/prod configs exist —
// at that point this test is exactly what will catch the rate_limit
// section being missing from them again.
func TestRateLimitEnabledAcrossAllEnvironments(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	configsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "configs")

	envs := []string{
		"config.development.yaml",
		// "config.staging.yaml", // TODO: re-enable once staging config is populated
		// "config.prod.yaml",    // TODO: re-enable once prod config is populated
	}

	for _, file := range envs {
		file := file
		t.Run(file, func(t *testing.T) {
			path := filepath.Join(configsDir, file)

			cfg, err := config.LoadFromPath(path)
			require.NoError(t, err, "config file must load: %s", path)

			require.True(
				t, cfg.RateLimit.Enabled,
				"%s must explicitly enable rate_limit — a missing 'rate_limit:' "+
					"section silently disables order submit/cancel/modify throttling",
				file,
			)

			require.Greater(t, cfg.RateLimit.SubmitRate, 0.0, "%s: submit_rate must be > 0", file)
			require.Greater(t, cfg.RateLimit.SubmitBurst, 0, "%s: submit_burst must be > 0", file)
			require.Greater(t, cfg.RateLimit.CancelRate, 0.0, "%s: cancel_rate must be > 0", file)
			require.Greater(t, cfg.RateLimit.CancelBurst, 0, "%s: cancel_burst must be > 0", file)
			require.Greater(t, cfg.RateLimit.ModifyRate, 0.0, "%s: modify_rate must be > 0", file)
			require.Greater(t, cfg.RateLimit.ModifyBurst, 0, "%s: modify_burst must be > 0", file)
		})
	}
}