package gateway

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/gateway"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("Without debug endpoints", func(t *testing.T) {
		cfg := DefaultConfig(false, true, true)
		assert.NotNil(t, cfg.V1PbMux.Mux)
		require.Equal(t, 1, len(cfg.V1PbMux.Patterns))
		assert.Equal(t, "/eth/v1/", cfg.V1PbMux.Patterns[0])
		assert.Equal(t, 4, len(cfg.V1PbMux.Registrations))
		assert.NotNil(t, cfg.V1Alpha1PbMux.Mux)
		require.Equal(t, 1, len(cfg.V1Alpha1PbMux.Patterns))
		assert.Equal(t, "/eth/v1alpha1/", cfg.V1Alpha1PbMux.Patterns[0])
		assert.Equal(t, 4, len(cfg.V1Alpha1PbMux.Registrations))
	})

	t.Run("With debug endpoints", func(t *testing.T) {
		cfg := DefaultConfig(true, true, true)
		assert.NotNil(t, cfg.V1PbMux.Mux)
		require.Equal(t, 1, len(cfg.V1PbMux.Patterns))
		assert.Equal(t, "/eth/v1/", cfg.V1PbMux.Patterns[0])
		assert.Equal(t, 5, len(cfg.V1PbMux.Registrations))
		assert.NotNil(t, cfg.V1Alpha1PbMux.Mux)
		require.Equal(t, 1, len(cfg.V1Alpha1PbMux.Patterns))
		assert.Equal(t, "/eth/v1alpha1/", cfg.V1Alpha1PbMux.Patterns[0])
		assert.Equal(t, 5, len(cfg.V1Alpha1PbMux.Registrations))
	})
	t.Run("Without Prysm API", func(t *testing.T) {
		cfg := DefaultConfig(true, false, true)
		assert.NotNil(t, cfg.V1PbMux.Mux)
		require.Equal(t, 1, len(cfg.V1PbMux.Patterns))
		assert.Equal(t, "/eth/v1/", cfg.V1PbMux.Patterns[0])
		assert.Equal(t, 5, len(cfg.V1PbMux.Registrations))
		assert.Equal(t, (*gateway.PbMux)(nil), cfg.V1Alpha1PbMux)
	})
	t.Run("Without Eth API", func(t *testing.T) {
		cfg := DefaultConfig(true, true, false)
		assert.Equal(t, (*gateway.PbMux)(nil), cfg.V1PbMux)
		assert.NotNil(t, cfg.V1Alpha1PbMux.Mux)
		require.Equal(t, 1, len(cfg.V1Alpha1PbMux.Patterns))
		assert.Equal(t, "/eth/v1alpha1/", cfg.V1Alpha1PbMux.Patterns[0])
		assert.Equal(t, 5, len(cfg.V1Alpha1PbMux.Registrations))
	})
}
