// Package tts provides a Twilio implementation of tts.Provider.
//
// Twilio TTS works differently from other providers - instead of returning
// audio bytes, it plays audio on calls via TwiML. This provider supports
// both TwiML generation and direct audio synthesis (using Twilio's
// text-to-speech capabilities).
package tts

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"sync"

	"github.com/plexusone/omnivoice-core/tts"
)

// Verify interface compliance at compile time.
var _ tts.Provider = (*Provider)(nil)

// Provider implements tts.Provider using Twilio's TTS capabilities.
type Provider struct {
	defaultVoice    string
	defaultLanguage string

	mu          sync.RWMutex
	voicesCache []tts.Voice
}

// Option configures the Provider.
type Option func(*options)

type options struct {
	voice    string
	language string
}

// WithVoice sets the default voice.
func WithVoice(voice string) Option {
	return func(o *options) {
		o.voice = voice
	}
}

// WithLanguage sets the default language.
func WithLanguage(language string) Option {
	return func(o *options) {
		o.language = language
	}
}

// New creates a new Twilio TTS provider.
func New(opts ...Option) (*Provider, error) {
	cfg := &options{
		voice:    "alice",
		language: "en-US",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Provider{
		defaultVoice:    cfg.voice,
		defaultLanguage: cfg.language,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "twilio"
}

// Synthesize converts text to speech.
// Note: Twilio TTS is designed for use within calls via TwiML.
// This method generates TwiML that can be used with Twilio's API.
func (p *Provider) Synthesize(ctx context.Context, text string, config tts.SynthesisConfig) (*tts.SynthesisResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Generate TwiML
	twiml := p.generateTwiML(text, config)

	// Return TwiML as "audio" - this is a special case for Twilio
	// The caller should use this TwiML with Twilio's API
	return &tts.SynthesisResult{
		Audio:          []byte(twiml),
		Format:         "twiml",
		CharacterCount: len(text),
	}, nil
}

// SynthesizeStream is not directly supported by Twilio.
// For streaming, use Media Streams with ElevenLabs or another provider.
func (p *Provider) SynthesizeStream(ctx context.Context, text string, config tts.SynthesisConfig) (<-chan tts.StreamChunk, error) {
	out := make(chan tts.StreamChunk, 1)

	go func() {
		defer close(out)

		// Generate TwiML
		twiml := p.generateTwiML(text, config)
		out <- tts.StreamChunk{
			Audio:   []byte(twiml),
			IsFinal: true,
		}
	}()

	return out, nil
}

// ListVoices returns available Twilio voices.
func (p *Provider) ListVoices(ctx context.Context) ([]tts.Voice, error) {
	p.mu.RLock()
	if p.voicesCache != nil {
		cached := make([]tts.Voice, len(p.voicesCache))
		copy(cached, p.voicesCache)
		p.mu.RUnlock()
		return cached, nil
	}
	p.mu.RUnlock()

	voices := getTwilioVoices()

	p.mu.Lock()
	p.voicesCache = voices
	p.mu.Unlock()

	return voices, nil
}

// GetVoice returns a specific voice by ID.
func (p *Provider) GetVoice(ctx context.Context, voiceID string) (*tts.Voice, error) {
	voices, err := p.ListVoices(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range voices {
		if v.ID == voiceID {
			return &v, nil
		}
	}

	return nil, tts.ErrVoiceNotFound
}

// GenerateTwiML generates TwiML for text-to-speech.
// This can be used directly with Twilio's API.
func (p *Provider) GenerateTwiML(text string, config tts.SynthesisConfig) string {
	return p.generateTwiML(text, config)
}

// SayElement represents a TwiML <Say> element.
type SayElement struct {
	XMLName  xml.Name `xml:"Say"`
	Voice    string   `xml:"voice,attr,omitempty"`
	Language string   `xml:"language,attr,omitempty"`
	Text     string   `xml:",chardata"`
}

// ResponseElement represents a TwiML <Response> element.
type ResponseElement struct {
	XMLName xml.Name `xml:"Response"`
	Say     *SayElement
}

func (p *Provider) generateTwiML(text string, config tts.SynthesisConfig) string {
	voice := config.VoiceID
	if voice == "" {
		voice = p.defaultVoice
	}

	language := p.defaultLanguage
	if config.Model != "" {
		// Use model as language override
		language = config.Model
	}

	say := &SayElement{
		Voice:    voice,
		Language: language,
		Text:     text,
	}

	response := &ResponseElement{Say: say}

	xmlBytes, err := xml.MarshalIndent(response, "", "    ")
	if err != nil {
		return fmt.Sprintf(`<Response><Say>%s</Say></Response>`, text)
	}

	return xml.Header + string(xmlBytes)
}

// SynthesizeToWriter generates TwiML and writes it to a writer.
func (p *Provider) SynthesizeToWriter(ctx context.Context, text string, config tts.SynthesisConfig, w io.Writer) error {
	twiml := p.generateTwiML(text, config)
	_, err := w.Write([]byte(twiml))
	return err
}

// getTwilioVoices returns the available Twilio voices.
func getTwilioVoices() []tts.Voice {
	return []tts.Voice{
		// Basic Twilio voices
		{ID: "alice", Name: "Alice", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "man", Name: "Man", Language: "en-US", Gender: "male", Provider: "twilio"},
		{ID: "woman", Name: "Woman", Language: "en-US", Gender: "female", Provider: "twilio"},

		// Amazon Polly voices (require Polly. prefix)
		{ID: "Polly.Joanna", Name: "Joanna (Polly)", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Matthew", Name: "Matthew (Polly)", Language: "en-US", Gender: "male", Provider: "twilio"},
		{ID: "Polly.Amy", Name: "Amy (Polly)", Language: "en-GB", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Brian", Name: "Brian (Polly)", Language: "en-GB", Gender: "male", Provider: "twilio"},
		{ID: "Polly.Ivy", Name: "Ivy (Polly)", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Kendra", Name: "Kendra (Polly)", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Kimberly", Name: "Kimberly (Polly)", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Salli", Name: "Salli (Polly)", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Joey", Name: "Joey (Polly)", Language: "en-US", Gender: "male", Provider: "twilio"},
		{ID: "Polly.Justin", Name: "Justin (Polly)", Language: "en-US", Gender: "male", Provider: "twilio"},

		// Google TTS voices (require Google. prefix)
		{ID: "Google.en-US-Standard-A", Name: "Google US Female A", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Google.en-US-Standard-B", Name: "Google US Male B", Language: "en-US", Gender: "male", Provider: "twilio"},
		{ID: "Google.en-US-Standard-C", Name: "Google US Female C", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Google.en-US-Standard-D", Name: "Google US Male D", Language: "en-US", Gender: "male", Provider: "twilio"},
		{ID: "Google.en-US-Wavenet-A", Name: "Google US Wavenet Female A", Language: "en-US", Gender: "female", Provider: "twilio"},
		{ID: "Google.en-US-Wavenet-B", Name: "Google US Wavenet Male B", Language: "en-US", Gender: "male", Provider: "twilio"},

		// Spanish voices
		{ID: "Polly.Penelope", Name: "Penelope (Polly)", Language: "es-US", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Miguel", Name: "Miguel (Polly)", Language: "es-US", Gender: "male", Provider: "twilio"},

		// French voices
		{ID: "Polly.Celine", Name: "Celine (Polly)", Language: "fr-FR", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Mathieu", Name: "Mathieu (Polly)", Language: "fr-FR", Gender: "male", Provider: "twilio"},

		// German voices
		{ID: "Polly.Marlene", Name: "Marlene (Polly)", Language: "de-DE", Gender: "female", Provider: "twilio"},
		{ID: "Polly.Hans", Name: "Hans (Polly)", Language: "de-DE", Gender: "male", Provider: "twilio"},
	}
}
