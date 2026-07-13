package registry_test

import (
	"sync"
	"testing"

	"velocity/internal/engine/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryCreatesEngine(t *testing.T) {
	r := registry.New(nil)

	e := r.Get("BTCUSDT")

	require.NotNil(t, e)

	assert.Equal(
		t,
		1,
		r.Count(),
	)
}

func TestRegistryReturnsSameEngine(t *testing.T) {
	r := registry.New(nil)

	e1 := r.Get("BTCUSDT")
	e2 := r.Get("BTCUSDT")

	assert.Same(
		t,
		e1,
		e2,
	)
}

func TestRegistryCreatesDifferentEngines(t *testing.T) {
	r := registry.New(nil)

	btc := r.Get("BTCUSDT")
	eth := r.Get("ETHUSDT")

	assert.NotSame(
		t,
		btc,
		eth,
	)

	assert.Equal(
		t,
		2,
		r.Count(),
	)
}

func TestRegistryExists(t *testing.T) {
	r := registry.New(nil)

	assert.False(
		t,
		r.Exists("BTCUSDT"),
	)

	r.Get("BTCUSDT")

	assert.True(
		t,
		r.Exists("BTCUSDT"),
	)
}

func TestRegistryRemove(t *testing.T) {
	r := registry.New(nil)

	r.Get("BTCUSDT")

	assert.True(
		t,
		r.Exists("BTCUSDT"),
	)

	r.Remove("BTCUSDT")

	assert.False(
		t,
		r.Exists("BTCUSDT"),
	)

	assert.Equal(
		t,
		0,
		r.Count(),
	)
}

func TestRegistrySymbols(t *testing.T) {
	r := registry.New(nil)

	r.Get("BTCUSDT")
	r.Get("ETHUSDT")
	r.Get("SOLUSDT")

	symbols := r.Symbols()

	assert.Len(
		t,
		symbols,
		3,
	)

	assert.Contains(
		t,
		symbols,
		"BTCUSDT",
	)

	assert.Contains(
		t,
		symbols,
		"ETHUSDT",
	)

	assert.Contains(
		t,
		symbols,
		"SOLUSDT",
	)
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := registry.New(nil)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			r.Get("BTCUSDT")
		}()
	}

	wg.Wait()

	assert.Equal(
		t,
		1,
		r.Count(),
	)
}

func TestRegistryConcurrentDifferentSymbols(
	t *testing.T,
) {
	r := registry.New(nil)

	symbols := []string{
		"BTCUSDT",
		"ETHUSDT",
		"SOLUSDT",
		"BNBUSDT",
		"ADAUSDT",
	}

	var wg sync.WaitGroup

	for _, symbol := range symbols {
		wg.Add(1)

		go func(s string) {
			defer wg.Done()

			r.Get(s)
		}(symbol)
	}

	wg.Wait()

	assert.Equal(
		t,
		len(symbols),
		r.Count(),
	)
}