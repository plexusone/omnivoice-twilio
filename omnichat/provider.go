// Package omnichat provides a Twilio SMS provider for omnichat.
package omnichat

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/plexusone/omnichat/provider"

	"github.com/plexusone/twilio-go/client"
)

// Verify interface compliance at compile time.
var _ provider.Provider = (*Provider)(nil)

// Provider implements provider.Provider for Twilio SMS.
type Provider struct {
	client         *client.Client
	defaultFrom    string
	logger         *slog.Logger
	messageHandler provider.MessageHandler
	eventHandler   provider.EventHandler
	webhookHandler http.Handler

	mu        sync.RWMutex
	connected bool
}

// Option configures the Provider.
type Option func(*options)

type options struct {
	accountSID  string
	authToken   string
	phoneNumber string
	logger      *slog.Logger
}

// WithAccountSID sets the Twilio Account SID.
func WithAccountSID(sid string) Option {
	return func(o *options) {
		o.accountSID = sid
	}
}

// WithAuthToken sets the Twilio Auth Token.
func WithAuthToken(token string) Option {
	return func(o *options) {
		o.authToken = token
	}
}

// WithPhoneNumber sets the default outbound phone number.
func WithPhoneNumber(number string) Option {
	return func(o *options) {
		o.phoneNumber = number
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// New creates a new Twilio SMS provider.
func New(opts ...Option) (*Provider, error) {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.logger == nil {
		cfg.logger = slog.Default()
	}

	twilioClient, err := client.New(&client.Config{
		AccountSID: cfg.accountSID,
		AuthToken:  cfg.authToken,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Twilio client: %w", err)
	}

	p := &Provider{
		client:      twilioClient,
		defaultFrom: cfg.phoneNumber,
		logger:      cfg.logger,
	}

	// Create webhook handler
	p.webhookHandler = http.HandlerFunc(p.handleWebhook)

	return p, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "twilio-sms"
}

// Connect establishes connection to Twilio.
// For SMS, this validates credentials but doesn't maintain a persistent connection.
func (p *Provider) Connect(ctx context.Context) error {
	// Validate credentials by listing phone numbers
	_, err := p.client.ListPhoneNumbers(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate Twilio credentials: %w", err)
	}

	p.mu.Lock()
	p.connected = true
	p.mu.Unlock()

	p.logger.Info("twilio SMS provider connected")
	return nil
}

// Disconnect closes the Twilio connection.
func (p *Provider) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	p.connected = false
	p.mu.Unlock()

	p.logger.Info("twilio SMS provider disconnected")
	return nil
}

// Send sends an SMS message.
// The chatID is the recipient phone number in E.164 format (e.g., "+1234567890").
func (p *Provider) Send(ctx context.Context, chatID string, msg provider.OutgoingMessage) error {
	p.mu.RLock()
	connected := p.connected
	p.mu.RUnlock()

	if !connected {
		return fmt.Errorf("provider not connected")
	}

	from := p.defaultFrom
	if from == "" {
		return fmt.Errorf("from phone number not configured")
	}

	twilioMsg, err := p.client.SendSMS(ctx, &client.SendSMSParams{
		To:   chatID,
		From: from,
		Body: msg.Content,
	})
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	p.logger.Debug("SMS sent",
		"to", chatID,
		"from", from,
		"sid", twilioMsg.SID,
		"status", twilioMsg.Status,
	)

	return nil
}

// OnMessage registers a handler for incoming messages.
func (p *Provider) OnMessage(handler provider.MessageHandler) {
	p.mu.Lock()
	p.messageHandler = handler
	p.mu.Unlock()
}

// OnEvent registers a handler for events.
func (p *Provider) OnEvent(handler provider.EventHandler) {
	p.mu.Lock()
	p.eventHandler = handler
	p.mu.Unlock()
}

// WebhookHandler returns an HTTP handler for Twilio webhooks.
// This should be mounted at a publicly accessible URL and configured
// in your Twilio console for incoming message webhooks.
func (p *Provider) WebhookHandler() http.Handler {
	return p.webhookHandler
}

// handleWebhook processes incoming Twilio webhooks.
func (p *Provider) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size to prevent memory exhaustion (G120 fix)
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10) // 64KB - Twilio webhooks are typically small
	if err := r.ParseForm(); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract message data from Twilio webhook (use r.Form.Get per G120)
	messageSID := r.Form.Get("MessageSid")
	from := r.Form.Get("From")
	to := r.Form.Get("To")
	body := r.Form.Get("Body")
	accountSID := r.Form.Get("AccountSid")

	if messageSID == "" || from == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	p.mu.RLock()
	handler := p.messageHandler
	p.mu.RUnlock()

	if handler != nil {
		msg := provider.IncomingMessage{
			ID:           messageSID,
			ProviderName: "twilio-sms",
			ChatID:       from,
			ChatType:     provider.ChatTypeDM,
			SenderID:     from,
			SenderName:   from,
			Content:      body,
			Timestamp:    time.Now(),
			Metadata: map[string]any{
				"to":           to,
				"account_sid":  accountSID,
				"num_media":    r.Form.Get("NumMedia"),
				"from_city":    r.Form.Get("FromCity"),
				"from_state":   r.Form.Get("FromState"),
				"from_zip":     r.Form.Get("FromZip"),
				"from_country": r.Form.Get("FromCountry"),
			},
		}

		ctx := r.Context()
		if err := handler(ctx, msg); err != nil {
			p.logger.Error("message handler failed", "error", err, "sid", messageSID)
		}
	}

	// Return empty TwiML response
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Response></Response>`))
}

// SendFrom sends an SMS from a specific phone number.
func (p *Provider) SendFrom(ctx context.Context, to, from, body string) error {
	twilioMsg, err := p.client.SendSMS(ctx, &client.SendSMSParams{
		To:   to,
		From: from,
		Body: body,
	})
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	p.logger.Debug("SMS sent",
		"to", to,
		"from", from,
		"sid", twilioMsg.SID,
		"status", twilioMsg.Status,
	)

	return nil
}
