// Package stt provides a Twilio implementation of stt.Provider.
//
// Twilio STT works within call contexts via TwiML <Gather> verb or
// real-time transcription on Media Streams. This provider supports
// TwiML generation and real-time transcription configuration.
package stt

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/plexusone/omnivoice-core/stt"
)

// Verify interface compliance at compile time.
var (
	_ stt.Provider          = (*Provider)(nil)
	_ stt.StreamingProvider = (*Provider)(nil)
)

// Provider implements stt.Provider using Twilio's speech recognition.
type Provider struct {
	defaultLanguage string
	speechModel     string
	profanityFilter bool
}

// Option configures the Provider.
type Option func(*options)

type options struct {
	language        string
	speechModel     string
	profanityFilter bool
}

// WithLanguage sets the default language.
func WithLanguage(language string) Option {
	return func(o *options) {
		o.language = language
	}
}

// WithSpeechModel sets the speech recognition model.
// Options: "default", "numbers_and_commands", "phone_call", "video", "enhanced"
func WithSpeechModel(model string) Option {
	return func(o *options) {
		o.speechModel = model
	}
}

// WithProfanityFilter enables or disables the profanity filter.
func WithProfanityFilter(enabled bool) Option {
	return func(o *options) {
		o.profanityFilter = enabled
	}
}

// New creates a new Twilio STT provider.
func New(opts ...Option) (*Provider, error) {
	cfg := &options{
		language:        "en-US",
		speechModel:     "phone_call",
		profanityFilter: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Provider{
		defaultLanguage: cfg.language,
		speechModel:     cfg.speechModel,
		profanityFilter: cfg.profanityFilter,
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "twilio"
}

// Transcribe is not directly supported for arbitrary audio.
// Twilio STT works within call contexts.
func (p *Provider) Transcribe(ctx context.Context, audio []byte, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return nil, fmt.Errorf("direct audio transcription not supported; use TranscribeStream within a call or use ElevenLabs/other provider")
}

// TranscribeFile is not supported by Twilio.
func (p *Provider) TranscribeFile(ctx context.Context, filePath string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return nil, fmt.Errorf("file transcription not supported; Twilio STT works within call contexts")
}

// TranscribeURL is not supported by Twilio.
func (p *Provider) TranscribeURL(ctx context.Context, url string, config stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return nil, fmt.Errorf("URL transcription not supported; Twilio STT works within call contexts")
}

// TranscribeStream creates a streaming transcription session.
// This works with Twilio's real-time transcription on Media Streams.
func (p *Provider) TranscribeStream(ctx context.Context, config stt.TranscriptionConfig) (io.WriteCloser, <-chan stt.StreamEvent, error) {
	eventCh := make(chan stt.StreamEvent, 100)
	writer := &streamWriter{
		provider: p,
		config:   config,
		eventCh:  eventCh,
		ctx:      ctx,
	}

	return writer, eventCh, nil
}

// streamWriter implements io.WriteCloser for streaming transcription.
type streamWriter struct {
	provider *Provider
	config   stt.TranscriptionConfig
	eventCh  chan stt.StreamEvent
	ctx      context.Context
	mu       sync.Mutex
	closed   bool
}

// Write processes incoming audio and emits transcription events.
// Note: This is a placeholder - actual Twilio transcription happens
// on their servers via Media Streams with real-time transcription enabled.
func (w *streamWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, io.ErrClosedPipe
	}

	// In a real implementation, this audio would be processed by Twilio's
	// real-time transcription service via Media Streams.
	// The transcription results come back through the WebSocket connection.
	return len(p), nil
}

// Close closes the stream.
func (w *streamWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.closed {
		w.closed = true
		close(w.eventCh)
	}
	return nil
}

// GenerateGatherTwiML generates TwiML for speech recognition.
// Use this to create interactive voice response (IVR) flows.
func (p *Provider) GenerateGatherTwiML(config GatherConfig) string {
	gather := &GatherElement{
		Input:           config.Input,
		Language:        config.Language,
		SpeechTimeout:   config.SpeechTimeout,
		Timeout:         config.Timeout,
		NumDigits:       config.NumDigits,
		FinishOnKey:     config.FinishOnKey,
		Action:          config.Action,
		Method:          config.Method,
		Enhanced:        config.Enhanced,
		SpeechModel:     config.SpeechModel,
		ProfanityFilter: config.ProfanityFilter,
	}

	if config.Prompt != "" {
		gather.Say = &SayElement{Text: config.Prompt}
	}

	response := &ResponseElement{Gather: gather}

	xmlBytes, err := xml.MarshalIndent(response, "", "    ")
	if err != nil {
		return fmt.Sprintf(`<Response><Gather input="%s"/></Response>`, config.Input)
	}

	return xml.Header + string(xmlBytes)
}

// GenerateRealTimeTranscriptionConfig generates the configuration
// for enabling real-time transcription on Twilio Media Streams.
func (p *Provider) GenerateRealTimeTranscriptionConfig() map[string]any {
	return map[string]any{
		"transcription": map[string]any{
			"mode":        "real-time",
			"language":    p.defaultLanguage,
			"speechModel": p.speechModel,
			"profanityFilter": map[string]bool{
				"enabled": p.profanityFilter,
			},
		},
	}
}

// GatherConfig configures a TwiML <Gather> element.
type GatherConfig struct {
	// Input specifies input types: "speech", "dtmf", or "speech dtmf"
	Input string

	// Language is the speech recognition language (e.g., "en-US")
	Language string

	// SpeechTimeout is seconds of silence before finalizing speech
	SpeechTimeout string

	// Timeout is seconds to wait for input
	Timeout int

	// NumDigits is max DTMF digits to collect
	NumDigits int

	// FinishOnKey is the DTMF key that ends input
	FinishOnKey string

	// Action is the URL to submit results to
	Action string

	// Method is the HTTP method ("GET" or "POST")
	Method string

	// Enhanced enables enhanced speech recognition
	Enhanced bool

	// SpeechModel is the speech model to use
	SpeechModel string

	// ProfanityFilter filters profanity
	ProfanityFilter bool

	// Prompt is the text to say before gathering
	Prompt string
}

// GatherElement represents a TwiML <Gather> element.
type GatherElement struct {
	XMLName         xml.Name    `xml:"Gather"`
	Input           string      `xml:"input,attr,omitempty"`
	Language        string      `xml:"language,attr,omitempty"`
	SpeechTimeout   string      `xml:"speechTimeout,attr,omitempty"`
	Timeout         int         `xml:"timeout,attr,omitempty"`
	NumDigits       int         `xml:"numDigits,attr,omitempty"`
	FinishOnKey     string      `xml:"finishOnKey,attr,omitempty"`
	Action          string      `xml:"action,attr,omitempty"`
	Method          string      `xml:"method,attr,omitempty"`
	Enhanced        bool        `xml:"enhanced,attr,omitempty"`
	SpeechModel     string      `xml:"speechModel,attr,omitempty"`
	ProfanityFilter bool        `xml:"profanityFilter,attr,omitempty"`
	Say             *SayElement `xml:",omitempty"`
}

// SayElement represents a TwiML <Say> element.
type SayElement struct {
	XMLName xml.Name `xml:"Say"`
	Text    string   `xml:",chardata"`
}

// ResponseElement represents a TwiML <Response> element.
type ResponseElement struct {
	XMLName xml.Name       `xml:"Response"`
	Gather  *GatherElement `xml:",omitempty"`
}

// TranscriptionEvent represents a real-time transcription event from Twilio.
type TranscriptionEvent struct {
	Type       string    `json:"type"`
	Transcript string    `json:"transcript"`
	Confidence float64   `json:"confidence"`
	IsFinal    bool      `json:"is_final"`
	Language   string    `json:"language"`
	StartTime  float64   `json:"start_time"`
	EndTime    float64   `json:"end_time"`
	Timestamp  time.Time `json:"timestamp"`
}

// ParseTranscriptionEvent parses a real-time transcription event from JSON.
func ParseTranscriptionEvent(data []byte) (*TranscriptionEvent, error) {
	var event TranscriptionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ToStreamEvent converts a Twilio transcription event to an OmniVoice stream event.
func (e *TranscriptionEvent) ToStreamEvent() stt.StreamEvent {
	event := stt.StreamEvent{
		Type:       stt.EventTranscript,
		Transcript: e.Transcript,
		IsFinal:    e.IsFinal,
	}

	if e.IsFinal {
		event.Segment = &stt.Segment{
			Text:       e.Transcript,
			Confidence: e.Confidence,
			Language:   e.Language,
			StartTime:  time.Duration(e.StartTime * float64(time.Second)),
			EndTime:    time.Duration(e.EndTime * float64(time.Second)),
		}
	}

	return event
}
