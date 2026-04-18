//go:build integration

package tts

import (
	"testing"

	"github.com/plexusone/omnivoice-core/tts/providertest"
)

func TestConformance(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}

	providertest.RunAll(t, providertest.Config{
		Provider:        p,
		SkipIntegration: false,
		TestVoiceID:     "alice",
	})
}
