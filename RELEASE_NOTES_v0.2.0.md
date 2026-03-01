# Release Notes: v0.2.0

**Release Date:** 2026-02-28

## Summary

Organization rename from agentplexus to plexusone with updated module path.

## Breaking Changes

| Component | Before | After |
|-----------|--------|-------|
| Go module | `github.com/agentplexus/omnivoice-twilio` | `github.com/plexusone/omnivoice-twilio` |

## Migration Guide

Update your import paths:

```go
// Before
import "github.com/agentplexus/omnivoice-twilio/callsystem"
import "github.com/agentplexus/omnivoice-twilio/transport"

// After
import "github.com/plexusone/omnivoice-twilio/callsystem"
import "github.com/plexusone/omnivoice-twilio/transport"
```

Update your `go.mod`:

```bash
go mod edit -droprequire github.com/agentplexus/omnivoice-twilio
go get github.com/plexusone/omnivoice-twilio@v0.2.0
go mod tidy
```

## Tests

- Added conformance tests for CallSystem, STT, TTS, and Transport providers

## Dependencies

- Updated to `github.com/plexusone/omnivoice-core` v0.5.0 (was `github.com/agentplexus/omnivoice`)
