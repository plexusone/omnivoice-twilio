//go:build integration

package stt

import (
	"testing"

	"github.com/plexusone/omnivoice-core/stt/providertest"
)

func TestConformance(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}

	providertest.RunAll(t, providertest.Config{
		Provider:          p,
		StreamingProvider: p,
		SkipIntegration:   true,
	})
}
