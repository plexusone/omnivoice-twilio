// Package twilio provides OmniVoice provider implementations using the Twilio API.
//
// This package implements OmniVoice interfaces for voice applications:
//   - callsystem.CallSystem: PSTN call handling (incoming/outgoing)
//   - transport.Transport: Twilio Media Streams for real-time audio
//   - tts.Provider: Text-to-speech via Twilio's Say verb
//   - stt.Provider: Speech-to-text via Twilio's Gather/Recognition
//
// # Installation
//
//	go get github.com/plexusone/omnivoice-twilio
//
// # Environment Variables
//
//	TWILIO_ACCOUNT_SID - Your Twilio Account SID
//	TWILIO_AUTH_TOKEN  - Your Twilio Auth Token
//
// # Quick Start
//
//	import (
//	    "github.com/plexusone/omnivoice-twilio/callsystem"
//	    "github.com/plexusone/omnivoice-twilio/transport"
//	)
//
//	// Create call system for PSTN calls
//	cs, _ := callsystem.New()
//
//	// Create transport for audio streaming
//	tr, _ := transport.New()
package twilio

// Version is the SDK version.
const Version = "0.1.0"

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
