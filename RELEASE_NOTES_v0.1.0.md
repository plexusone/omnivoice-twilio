# Release Notes: v0.1.0

**Release Date:** 2025-12-28

## Summary

Initial release of omnivoice-twilio, providing Twilio integration for the OmniVoice voice pipeline framework.

## Features

### Twilio Client

- `MakeCall` - Initiate outbound calls
- `GetCall` - Retrieve call details
- `UpdateCall` - Modify active calls
- `HangupCall` - Terminate calls
- `ListPhoneNumbers` - List available phone numbers

### WebSocket Transport

- Twilio Media Streams WebSocket transport for real-time audio streaming
- Bidirectional audio support for voice agents

### OmniVoice Integration

- Call system provider implementing `callsystem.Provider` interface
- Transport provider implementing `transport.Provider` interface
- Seamless integration with OmniVoice STT/TTS pipelines

## Installation

```bash
go get github.com/agentplexus/omnivoice-twilio@v0.1.0
```
