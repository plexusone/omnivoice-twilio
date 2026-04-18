//go:build integration

package omnichat

import (
	"testing"

	"github.com/plexusone/omnichat/provider/providertest"
)

func TestConformance(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}

	providertest.TestProvider(t, p, true)
}
