// Package client provides a Twilio API client wrapper using the official twilio-go SDK.
package client

import (
	"context"
	"fmt"
	"os"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Client wraps the official Twilio SDK for internal use.
type Client struct {
	restClient *twilio.RestClient
	accountSID string
}

// Config configures the Twilio client.
type Config struct {
	AccountSID string
	AuthToken  string //nolint:gosec // G117: field intentionally stores credential
}

// New creates a new Twilio client using the official SDK.
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	accountSID := cfg.AccountSID
	if accountSID == "" {
		accountSID = os.Getenv("TWILIO_ACCOUNT_SID")
	}
	if accountSID == "" {
		return nil, fmt.Errorf("TWILIO_ACCOUNT_SID is required")
	}

	authToken := cfg.AuthToken
	if authToken == "" {
		authToken = os.Getenv("TWILIO_AUTH_TOKEN")
	}
	if authToken == "" {
		return nil, fmt.Errorf("TWILIO_AUTH_TOKEN is required")
	}

	restClient := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	return &Client{
		restClient: restClient,
		accountSID: accountSID,
	}, nil
}

// AccountSID returns the account SID.
func (c *Client) AccountSID() string {
	return c.accountSID
}

// RestClient returns the underlying Twilio REST client.
func (c *Client) RestClient() *twilio.RestClient {
	return c.restClient
}

// Call represents a Twilio call resource.
type Call struct {
	SID        string
	AccountSID string
	To         string
	From       string
	Status     string
	Direction  string
	Duration   string
	StartTime  string
	EndTime    string
}

// callFromAPI converts an API call response to our Call type.
func callFromAPI(apiCall *openapi.ApiV2010Call) *Call {
	call := &Call{}
	if apiCall.Sid != nil {
		call.SID = *apiCall.Sid
	}
	if apiCall.AccountSid != nil {
		call.AccountSID = *apiCall.AccountSid
	}
	if apiCall.To != nil {
		call.To = *apiCall.To
	}
	if apiCall.From != nil {
		call.From = *apiCall.From
	}
	if apiCall.Status != nil {
		call.Status = *apiCall.Status
	}
	if apiCall.Direction != nil {
		call.Direction = *apiCall.Direction
	}
	if apiCall.Duration != nil {
		call.Duration = *apiCall.Duration
	}
	if apiCall.StartTime != nil {
		call.StartTime = *apiCall.StartTime
	}
	if apiCall.EndTime != nil {
		call.EndTime = *apiCall.EndTime
	}
	return call
}

// MakeCallParams are parameters for making a call.
type MakeCallParams struct {
	To                  string
	From                string
	URL                 string            // TwiML URL
	Twiml               string            // Inline TwiML
	StatusCallback      string            // Webhook for status updates
	StatusCallbackEvent []string          // Events to receive
	MachineDetection    string            // "Enable" or "DetectMessageEnd"
	Timeout             int               // Ring timeout in seconds
	Record              bool              // Record the call
	RecordingChannels   string            // "mono" or "dual"
	CustomParameters    map[string]string // Custom parameters (unused with SDK)
}

// MakeCall initiates an outbound call.
func (c *Client) MakeCall(ctx context.Context, params *MakeCallParams) (*Call, error) {
	createParams := &openapi.CreateCallParams{}
	createParams.SetTo(params.To)
	createParams.SetFrom(params.From)

	if params.URL != "" {
		createParams.SetUrl(params.URL)
	}
	if params.Twiml != "" {
		createParams.SetTwiml(params.Twiml)
	}
	if params.StatusCallback != "" {
		createParams.SetStatusCallback(params.StatusCallback)
	}
	if len(params.StatusCallbackEvent) > 0 {
		createParams.SetStatusCallbackEvent(params.StatusCallbackEvent)
	}
	if params.MachineDetection != "" {
		createParams.SetMachineDetection(params.MachineDetection)
	}
	if params.Timeout > 0 {
		createParams.SetTimeout(params.Timeout)
	}
	if params.Record {
		createParams.SetRecord(true)
	}
	if params.RecordingChannels != "" {
		createParams.SetRecordingChannels(params.RecordingChannels)
	}

	apiCall, err := c.restClient.Api.CreateCall(createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create call: %w", err)
	}

	return callFromAPI(apiCall), nil
}

// GetCall retrieves a call by SID.
func (c *Client) GetCall(ctx context.Context, callSID string) (*Call, error) {
	apiCall, err := c.restClient.Api.FetchCall(callSID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch call: %w", err)
	}

	return callFromAPI(apiCall), nil
}

// UpdateCallParams are parameters for updating a call.
type UpdateCallParams struct {
	URL    string // New TwiML URL
	Twiml  string // Inline TwiML
	Status string // "completed" to hang up, "canceled" to cancel
}

// UpdateCall modifies an in-progress call.
func (c *Client) UpdateCall(ctx context.Context, callSID string, params *UpdateCallParams) (*Call, error) {
	updateParams := &openapi.UpdateCallParams{}

	if params.URL != "" {
		updateParams.SetUrl(params.URL)
	}
	if params.Twiml != "" {
		updateParams.SetTwiml(params.Twiml)
	}
	if params.Status != "" {
		updateParams.SetStatus(params.Status)
	}

	apiCall, err := c.restClient.Api.UpdateCall(callSID, updateParams)
	if err != nil {
		return nil, fmt.Errorf("failed to update call: %w", err)
	}

	return callFromAPI(apiCall), nil
}

// HangupCall ends a call.
func (c *Client) HangupCall(ctx context.Context, callSID string) (*Call, error) {
	return c.UpdateCall(ctx, callSID, &UpdateCallParams{Status: "completed"})
}

// Message represents an SMS/MMS message.
type Message struct {
	SID        string
	AccountSID string
	To         string
	From       string
	Body       string
	Status     string
	Direction  string
	DateSent   string
}

// messageFromAPI converts an API message response to our Message type.
func messageFromAPI(apiMsg *openapi.ApiV2010Message) *Message {
	msg := &Message{}
	if apiMsg.Sid != nil {
		msg.SID = *apiMsg.Sid
	}
	if apiMsg.AccountSid != nil {
		msg.AccountSID = *apiMsg.AccountSid
	}
	if apiMsg.To != nil {
		msg.To = *apiMsg.To
	}
	if apiMsg.From != nil {
		msg.From = *apiMsg.From
	}
	if apiMsg.Body != nil {
		msg.Body = *apiMsg.Body
	}
	if apiMsg.Status != nil {
		msg.Status = *apiMsg.Status
	}
	if apiMsg.Direction != nil {
		msg.Direction = *apiMsg.Direction
	}
	if apiMsg.DateSent != nil {
		msg.DateSent = *apiMsg.DateSent
	}
	return msg
}

// SendSMSParams are parameters for sending an SMS.
type SendSMSParams struct {
	To   string
	From string
	Body string
}

// SendSMS sends an SMS message.
func (c *Client) SendSMS(ctx context.Context, params *SendSMSParams) (*Message, error) {
	createParams := &openapi.CreateMessageParams{}
	createParams.SetTo(params.To)
	createParams.SetFrom(params.From)
	createParams.SetBody(params.Body)

	apiMsg, err := c.restClient.Api.CreateMessage(createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to send SMS: %w", err)
	}

	return messageFromAPI(apiMsg), nil
}

// PhoneNumber represents a Twilio phone number.
type PhoneNumber struct {
	SID          string
	PhoneNumber  string
	FriendlyName string
	VoiceCapable bool
	SMSCapable   bool
	MMSCapable   bool
}

// ListPhoneNumbers returns all phone numbers on the account.
func (c *Client) ListPhoneNumbers(ctx context.Context) ([]PhoneNumber, error) {
	apiNumbers, err := c.restClient.Api.ListIncomingPhoneNumber(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list phone numbers: %w", err)
	}

	numbers := make([]PhoneNumber, 0, len(apiNumbers))
	for _, n := range apiNumbers {
		pn := PhoneNumber{}
		if n.Sid != nil {
			pn.SID = *n.Sid
		}
		if n.PhoneNumber != nil {
			pn.PhoneNumber = *n.PhoneNumber
		}
		if n.FriendlyName != nil {
			pn.FriendlyName = *n.FriendlyName
		}
		if n.Capabilities != nil {
			pn.VoiceCapable = n.Capabilities.Voice
			pn.SMSCapable = n.Capabilities.Sms
			pn.MMSCapable = n.Capabilities.Mms
		}
		numbers = append(numbers, pn)
	}

	return numbers, nil
}
