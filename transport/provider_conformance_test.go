//go:build integration

package transport

import (
	"testing"

	"github.com/plexusone/omnivoice-core/transport/providertest"
)

func TestConformance(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}

	providertest.RunAll(t, providertest.Config{
		Provider:        p,
		SkipIntegration: true,
	})
}
