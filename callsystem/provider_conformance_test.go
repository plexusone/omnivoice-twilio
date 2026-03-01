//go:build integration

package callsystem

import (
	"os"
	"testing"

	"github.com/plexusone/omnivoice-core/callsystem/providertest"
)

func TestConformance(t *testing.T) {
	sid := os.Getenv("TWILIO_ACCOUNT_SID")
	token := os.Getenv("TWILIO_AUTH_TOKEN")
	from := os.Getenv("TWILIO_PHONE_NUMBER")
	to := os.Getenv("TWILIO_TO_NUMBER")
	if sid == "" || token == "" {
		t.Skip("TWILIO_ACCOUNT_SID/TWILIO_AUTH_TOKEN not set")
	}

	p, err := New(
		WithAccountSID(sid),
		WithAuthToken(token),
		WithPhoneNumber(from),
	)
	if err != nil {
		t.Fatal(err)
	}

	providertest.RunAll(t, providertest.Config{
		Provider:        p,
		SkipIntegration: from == "" || to == "",
		TestPhoneNumber: to,
		TestFromNumber:  from,
	})
}
