// Package twilio provides a Go SDK for Twilio with adapters for omnichat and omnivoice.
//
// This package implements:
//   - omnichat.Provider: SMS messaging
//   - omnivoice interfaces: callsystem, transport, tts, stt
//
// # Installation
//
//	go get github.com/plexusone/twilio-go
//
// # Environment Variables
//
//	TWILIO_ACCOUNT_SID - Your Twilio Account SID
//	TWILIO_AUTH_TOKEN  - Your Twilio Auth Token
//
// # Quick Start - SMS (omnichat)
//
//	import "github.com/plexusone/twilio-go/omnichat"
//
//	provider, _ := omnichat.New(
//	    omnichat.WithPhoneNumber("+1234567890"),
//	)
//	provider.Send(ctx, "+1987654321", provider.OutgoingMessage{Content: "Hello!"})
//
// # Quick Start - Voice (omnivoice)
//
//	import (
//	    "github.com/plexusone/twilio-go/omnivoice/callsystem"
//	    "github.com/plexusone/twilio-go/omnivoice/transport"
//	)
//
//	cs, _ := callsystem.New()
//	tr, _ := transport.New()
package twilio

// Version is the SDK version.
const Version = "0.4.0"

// ProviderName is the name used to identify this provider in OmniVoice.
const ProviderName = "twilio"

// Twilio API constants.
const (
	// DefaultAPIBaseURL is the Twilio REST API base URL.
	DefaultAPIBaseURL = "https://api.twilio.com/2010-04-01"

	// DefaultMediaStreamURL is the WebSocket URL format for Media Streams.
	// Format: wss://media-stream.twilio.com/v1/Accounts/{AccountSid}/Calls/{CallSid}/Media
	DefaultMediaStreamURL = "wss://media-stream.twilio.com"
)

// Audio format constants for Media Streams.
const (
	// AudioEncodingMulaw is the μ-law encoding (8-bit, 8kHz).
	AudioEncodingMulaw = "audio/x-mulaw"

	// AudioEncodingPCM is the PCM encoding (16-bit, 8kHz).
	AudioEncodingPCM = "audio/x-l16"

	// DefaultSampleRate is the default sample rate for Twilio audio (8kHz).
	DefaultSampleRate = 8000
)

// TwiML voice options.
const (
	VoiceAlice  = "alice"   // Twilio's default voice
	VoiceMan    = "man"     // Male voice
	VoiceWoman  = "woman"   // Female voice
	VoicePolly  = "Polly."  // Amazon Polly prefix (e.g., "Polly.Joanna")
	VoiceGoogle = "Google." // Google TTS prefix (e.g., "Google.en-US-Standard-A")
	VoiceAmazon = "Amazon." // Amazon prefix
)

// Call status constants.
const (
	CallStatusQueued     = "queued"
	CallStatusRinging    = "ringing"
	CallStatusInProgress = "in-progress"
	CallStatusCompleted  = "completed"
	CallStatusBusy       = "busy"
	CallStatusFailed     = "failed"
	CallStatusNoAnswer   = "no-answer"
	CallStatusCanceled   = "canceled"
)
