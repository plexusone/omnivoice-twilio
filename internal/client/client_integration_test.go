//go:build integration

package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

func requireEnv(t *testing.T) (sid, token, from, to string) {
	t.Helper()
	sid = os.Getenv("TWILIO_ACCOUNT_SID")
	token = os.Getenv("TWILIO_AUTH_TOKEN")
	from = os.Getenv("TWILIO_PHONE_NUMBER")
	to = os.Getenv("TWILIO_TO_NUMBER")
	if sid == "" || token == "" {
		t.Skip("TWILIO_ACCOUNT_SID/TWILIO_AUTH_TOKEN not set")
	}
	return sid, token, from, to
}

func TestClientNew(t *testing.T) {
	sid, token, _, _ := requireEnv(t)

	c, err := New(&Config{
		AccountSID: sid,
		AuthToken:  token,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if c.AccountSID() != sid {
		t.Errorf("AccountSID() = %q, want %q", c.AccountSID(), sid)
	}
}

func TestListPhoneNumbers(t *testing.T) {
	sid, token, _, _ := requireEnv(t)

	c, err := New(&Config{
		AccountSID: sid,
		AuthToken:  token,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	numbers, err := c.ListPhoneNumbers(ctx)
	if err != nil {
		t.Fatalf("ListPhoneNumbers() error: %v", err)
	}
	if len(numbers) == 0 {
		t.Fatal("ListPhoneNumbers() returned empty list; expected at least one number")
	}

	for i, n := range numbers {
		t.Logf("Phone[%d]: SID=%s Number=%s Name=%s Voice=%v SMS=%v",
			i, n.SID, n.PhoneNumber, n.FriendlyName, n.VoiceCapable, n.SMSCapable)
		if n.SID == "" {
			t.Errorf("Phone[%d].SID is empty", i)
		}
		if n.PhoneNumber == "" {
			t.Errorf("Phone[%d].PhoneNumber is empty", i)
		}
	}
}

func TestGetCall_NotFound(t *testing.T) {
	sid, token, _, _ := requireEnv(t)

	c, err := New(&Config{
		AccountSID: sid,
		AuthToken:  token,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = c.GetCall(ctx, "CA00000000000000000000000000000000")
	if err == nil {
		t.Error("GetCall(bogus SID) should return error")
	}
	t.Logf("GetCall(bogus) error: %v", err)
}

func TestMakeCallAndHangup(t *testing.T) {
	sid, token, from, to := requireEnv(t)
	if from == "" {
		t.Skip("TWILIO_PHONE_NUMBER not set")
	}
	if to == "" {
		t.Skip("TWILIO_TO_NUMBER not set")
	}

	c, err := New(&Config{
		AccountSID: sid,
		AuthToken:  token,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Make a call with a TwiML message the recipient can hear
	call, err := c.MakeCall(ctx, &MakeCallParams{
		To:   to,
		From: from,
		Twiml: `<Response>` +
			`<Say voice="Polly.Joanna">This is an integration test call from omnivoice twilio. If you can hear this, the test is working. Goodbye!</Say>` +
			`</Response>`,
	})
	if err != nil {
		t.Fatalf("MakeCall() error: %v", err)
	}
	if call.SID == "" {
		t.Fatal("MakeCall() returned empty SID")
	}

	t.Logf("MakeCall() created call SID=%s Status=%s", call.SID, call.Status)
	t.Logf("Waiting for call to complete (answer the phone to hear the message)...")

	// Poll call status until it reaches a terminal state
	var finalStatus string
	for {
		time.Sleep(2 * time.Second)

		fetched, err := c.GetCall(ctx, call.SID)
		if err != nil {
			t.Fatalf("GetCall() error: %v", err)
		}

		t.Logf("  Call status: %s", fetched.Status)

		switch fetched.Status {
		case "completed", "canceled", "busy", "no-answer", "failed":
			finalStatus = fetched.Status
		}
		if finalStatus != "" {
			break
		}
		if ctx.Err() != nil {
			// Timeout — cancel the call
			t.Log("Timeout reached, hanging up")
			_, _ = c.HangupCall(ctx, call.SID)
			finalStatus = "canceled"
			break
		}
	}

	t.Logf("Final call status: %s", finalStatus)
	if finalStatus == "completed" {
		t.Log("Call completed — recipient answered and heard the message")
	}
}

// TestRoundTripSTTTTS makes a call, asks you to speak, transcribes your speech,
// and reads it back to you using TTS.
//
// Requires:
//   - TWILIO_WEBHOOK_URL: publicly accessible base URL (e.g., https://xxxx.ngrok.io)
//   - A tunnel forwarding to the local port (e.g., ngrok http 8081)
//
// Setup:
//  1. Run: ngrok http 8081
//  2. Set: export TWILIO_WEBHOOK_URL="https://xxxx.ngrok.io"
//  3. Run: go test -v -tags=integration -run TestRoundTrip ./internal/client/
func TestRoundTripSTTTTS(t *testing.T) {
	sid, token, from, to := requireEnv(t)
	if from == "" {
		t.Skip("TWILIO_PHONE_NUMBER not set")
	}
	if to == "" {
		t.Skip("TWILIO_TO_NUMBER not set")
	}
	webhookURL := os.Getenv("TWILIO_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("TWILIO_WEBHOOK_URL not set (run ngrok and set this to the public URL)")
	}

	port := os.Getenv("TWILIO_WEBHOOK_PORT")
	if port == "" {
		port = "8081"
	}

	// Track what the caller said
	var mu sync.Mutex
	var spokenText string

	// Start local HTTP server to handle Twilio callbacks
	mux := http.NewServeMux()

	// Twilio POSTs here with the Gather result
	mux.HandleFunc("/gather-result", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Logf("ParseForm error: %v", err)
			return
		}

		speechResult := r.FormValue("SpeechResult")
		t.Logf("Twilio heard: %q", speechResult)

		mu.Lock()
		spokenText = speechResult
		mu.Unlock()

		// Respond with TwiML that echoes what they said
		w.Header().Set("Content-Type", "application/xml")
		echoTwiML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Say voice="Polly.Joanna">You said: %s</Say>
    <Pause length="1"/>
    <Say voice="Polly.Joanna">Thank you. Goodbye!</Say>
</Response>`, speechResult)
		fmt.Fprint(w, echoTwiML)
	})

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		t.Fatalf("Failed to listen on port %s: %v", port, err)
	}
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	t.Logf("Webhook server listening on :%s", port)

	// Create Twilio client
	c, err := New(&Config{
		AccountSID: sid,
		AuthToken:  token,
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// TwiML: greet, gather speech, send result to our webhook
	gatherTwiML := fmt.Sprintf(`<Response>
    <Gather input="speech" action="%s/gather-result" method="POST" speechTimeout="auto" language="en-US">
        <Say voice="Polly.Joanna">Hello! This is the omnivoice round trip test. Please say something after the beep, and I will repeat it back to you.</Say>
    </Gather>
    <Say voice="Polly.Joanna">I did not hear anything. Goodbye!</Say>
</Response>`, webhookURL)

	call, err := c.MakeCall(ctx, &MakeCallParams{
		To:    to,
		From:  from,
		Twiml: gatherTwiML,
	})
	if err != nil {
		t.Fatalf("MakeCall() error: %v", err)
	}

	t.Logf("Call started SID=%s — answer the phone and say something!", call.SID)

	// Poll until the call completes
	var finalStatus string
	for {
		time.Sleep(2 * time.Second)

		fetched, err := c.GetCall(ctx, call.SID)
		if err != nil {
			t.Fatalf("GetCall() error: %v", err)
		}
		t.Logf("  Call status: %s", fetched.Status)

		switch fetched.Status {
		case "completed", "canceled", "busy", "no-answer", "failed":
			finalStatus = fetched.Status
		}
		if finalStatus != "" {
			break
		}
		if ctx.Err() != nil {
			t.Log("Timeout reached, hanging up")
			_, _ = c.HangupCall(ctx, call.SID)
			finalStatus = "canceled"
			break
		}
	}

	t.Logf("Final call status: %s", finalStatus)

	mu.Lock()
	heard := spokenText
	mu.Unlock()

	if heard != "" {
		t.Logf("Round-trip successful! You said: %q", heard)
	} else if finalStatus == "completed" {
		t.Log("Call completed but no speech was captured (did you speak?)")
	}
}
