# OmniVoice Twilio Provider

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

Twilio provider implementation for [OmniVoice](https://github.com/agentplexus/omnivoice) - the voice abstraction layer for AgentPlexus.

## Features

- **CallSystem**: PSTN call handling (incoming/outgoing phone calls)
- **Transport**: Twilio Media Streams for real-time audio
- **TTS**: Text-to-speech via Twilio's Say verb (Alice, Polly, Google voices)
- **STT**: Speech recognition via Gather verb and real-time transcription

## Installation

```bash
go get github.com/agentplexus/omnivoice-twilio
```

## Quick Start

### Complete Voice Agent with Phone Calls

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"

    "github.com/agentplexus/omnivoice-twilio/callsystem"
    "github.com/agentplexus/omnivoice-twilio/transport"
)

func main() {
    // Create Twilio call system
    cs, err := callsystem.New(
        callsystem.WithPhoneNumber("+15551234567"),
        callsystem.WithWebhookURL("wss://your-server.com/media-stream"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Handle incoming calls
    cs.OnIncomingCall(func(call callsystem.Call) error {
        fmt.Printf("Incoming call from %s\n", call.From())
        return call.Answer(context.Background())
    })

    // Make outbound call
    call, err := cs.MakeCall(context.Background(), "+15559876543")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Call initiated: %s\n", call.ID())

    // Set up HTTP handlers for Twilio webhooks
    http.HandleFunc("/incoming", handleIncoming(cs))
    http.HandleFunc("/media-stream", handleMediaStream(cs.Transport()))
    http.ListenAndServe(":8080", nil)
}

func handleIncoming(cs *callsystem.Provider) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        r.ParseForm()
        callSID := r.FormValue("CallSid")
        from := r.FormValue("From")
        to := r.FormValue("To")

        _, twiml, err := cs.HandleIncomingWebhook(callSID, from, to)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }

        w.Header().Set("Content-Type", "application/xml")
        w.Write([]byte(twiml))
    }
}

func handleMediaStream(tr *transport.Provider) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tr.HandleWebSocket(w, r, "/media-stream")
    }
}
```

### TTS (Text-to-Speech)

```go
import "github.com/agentplexus/omnivoice-twilio/tts"

provider, _ := tts.New(
    tts.WithVoice("Polly.Joanna"),
    tts.WithLanguage("en-US"),
)

// Generate TwiML for a call
result, _ := provider.Synthesize(ctx, "Hello, how can I help you?", tts.SynthesisConfig{
    VoiceID: "Polly.Matthew",
})

// result.Audio contains TwiML:
// <?xml version="1.0"?>
// <Response>
//     <Say voice="Polly.Matthew" language="en-US">Hello, how can I help you?</Say>
// </Response>
```

### STT (Speech-to-Text)

```go
import "github.com/agentplexus/omnivoice-twilio/stt"

provider, _ := stt.New(
    stt.WithLanguage("en-US"),
    stt.WithSpeechModel("phone_call"),
)

// Generate TwiML for speech recognition
twiml := provider.GenerateGatherTwiML(stt.GatherConfig{
    Input:         "speech",
    Language:      "en-US",
    SpeechTimeout: "auto",
    Action:        "/handle-speech",
    Prompt:        "Please say your account number",
})
```

### Transport (Media Streams)

```go
import "github.com/agentplexus/omnivoice-twilio/transport"

tr, _ := transport.New()

// Start listening for Media Stream connections
connCh, _ := tr.Listen(ctx, "/media-stream")

// Handle connections
for conn := range connCh {
    go func(c transport.Connection) {
        // Read audio from caller
        audio := make([]byte, 1024)
        for {
            n, err := c.AudioOut().Read(audio)
            if err != nil {
                break
            }
            // Process audio with STT...

            // Send audio back to caller
            c.AudioIn().Write(responseAudio)
        }
    }(conn)
}
```

## Full Agent Stack

For a complete voice agent, combine Twilio (calls + transport) with ElevenLabs (high-quality TTS/STT):

```go
import (
    "github.com/agentplexus/omnivoice/tts"
    "github.com/agentplexus/omnivoice/stt"
    twiliocs "github.com/agentplexus/omnivoice-twilio/callsystem"
    twiliotransport "github.com/agentplexus/omnivoice-twilio/transport"
    eleventts "github.com/agentplexus/omnivoice-elevenlabs/tts"
    elevenstt "github.com/agentplexus/omnivoice-elevenlabs/stt"
)

// Phone handling: Twilio
callSystem, _ := twiliocs.New()
transport, _ := twiliotransport.New()

// High-quality voice: ElevenLabs
ttsProvider, _ := eleventts.New()
sttProvider, _ := elevenstt.New()

// Multi-provider clients with fallback
ttsClient := tts.NewClient(ttsProvider)
sttClient := stt.NewClient(sttProvider)
```

## Configuration

### Environment Variables

```bash
export TWILIO_ACCOUNT_SID="your-account-sid"
export TWILIO_AUTH_TOKEN="your-auth-token"
```

### Explicit Configuration

```go
provider, _ := callsystem.New(
    callsystem.WithAccountSID("ACxxxxxxxx"),
    callsystem.WithAuthToken("your-token"),
    callsystem.WithPhoneNumber("+15551234567"),
    callsystem.WithWebhookURL("wss://your-server.com/media-stream"),
)
```

## Available Voices

### Twilio Basic
- `alice` - Default female voice
- `man` - Male voice
- `woman` - Female voice

### Amazon Polly (via Twilio)
- `Polly.Joanna`, `Polly.Matthew`, `Polly.Amy`, `Polly.Brian`
- `Polly.Ivy`, `Polly.Kendra`, `Polly.Kimberly`, `Polly.Salli`
- `Polly.Joey`, `Polly.Justin`

### Google TTS (via Twilio)
- `Google.en-US-Standard-A` through `D`
- `Google.en-US-Wavenet-A` through `D`

## Testing

Tests use the [OmniVoice conformance test](https://github.com/agentplexus/omnivoice) framework and are gated behind the `integration` build tag.

### Run All Tests

```bash
export TWILIO_ACCOUNT_SID="ACxxxx"
export TWILIO_AUTH_TOKEN="xxxx"
export TWILIO_PHONE_NUMBER="+15551234567"   # Your Twilio number (caller ID)
export TWILIO_TO_NUMBER="+15559876543"      # Recipient number for call tests

go test -v -tags=integration ./...
```

### Interface & Behavior Tests Only (No Credentials)

TTS, STT, and transport interface/behavior tests run without credentials:

```bash
go test -v -tags=integration ./tts/ ./stt/ ./transport/
```

### Call Lifecycle Tests Only

```bash
export TWILIO_ACCOUNT_SID="ACxxxx"
export TWILIO_AUTH_TOKEN="xxxx"
export TWILIO_PHONE_NUMBER="+15551234567"
export TWILIO_TO_NUMBER="+15559876543"

go test -v -tags=integration -run TestMakeCall ./internal/client/
go test -v -tags=integration -run TestConformance/Integration ./callsystem/
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Phone Call Flow                      │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Caller ←→ Twilio PSTN ←→ Media Streams ←→ Your Server  │
│                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │ CallSystem  │    │  Transport  │    │   Agent     │  │
│  │  (calls)    │←──→│  (audio)    │←──→│  (TTS/STT)  │  │
│  └─────────────┘    └─────────────┘    └─────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Requirements

- Go 1.21+
- Twilio Account (Account SID + Auth Token)
- Public webhook URL for incoming calls
- WebSocket endpoint for Media Streams

## Related Packages

- [omnivoice](https://github.com/agentplexus/omnivoice) - Core interfaces
- [omnivoice-elevenlabs](https://github.com/agentplexus/omnivoice-elevenlabs) - ElevenLabs provider (recommended for TTS/STT quality)

## License

MIT

 [build-status-svg]: https://github.com/agentplexus/omnivoice-twilio/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/agentplexus/omnivoice-twilio/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/agentplexus/omnivoice-twilio/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/agentplexus/omnivoice-twilio/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/agentplexus/omnivoice-twilio
 [goreport-url]: https://goreportcard.com/report/github.com/agentplexus/omnivoice-twilio
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/agentplexus/omnivoice-twilio
 [docs-godoc-url]: https://pkg.go.dev/github.com/agentplexus/omnivoice-twilio
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/agentplexus/omnivoice-twilio/blob/master/LICENSE
 [used-by-svg]: https://sourcegraph.com/github.com/agentplexus/omnivoice-twilio/-/badge.svg
 [used-by-url]: https://sourcegraph.com/github.com/agentplexus/omnivoice-twilio?badge
