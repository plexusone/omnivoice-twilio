// Package transport provides a Twilio Media Streams implementation of transport.Transport.
package transport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/plexusone/omnivoice-core/transport"
)

// Verify interface compliance at compile time.
var (
	_ transport.Transport          = (*Provider)(nil)
	_ transport.TelephonyTransport = (*Provider)(nil)
)

// Provider implements transport.Transport using Twilio Media Streams.
type Provider struct {
	accountSID string
	authToken  string

	mu          sync.RWMutex
	connections map[string]*Connection
	listeners   map[string]chan transport.Connection
	dtmfHandler func(conn transport.Connection, digit string)
}

// Option configures the Provider.
type Option func(*options)

type options struct {
	accountSID string
	authToken  string
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

// New creates a new Twilio Media Streams transport provider.
func New(opts ...Option) (*Provider, error) {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Provider{
		accountSID:  cfg.accountSID,
		authToken:   cfg.authToken,
		connections: make(map[string]*Connection),
		listeners:   make(map[string]chan transport.Connection),
	}, nil
}

// Name returns the transport name.
func (p *Provider) Name() string {
	return "twilio-media-streams"
}

// Protocol returns the protocol type.
func (p *Provider) Protocol() string {
	return "websocket"
}

// Listen starts listening for incoming Media Stream connections.
// The addr should be the path to handle (e.g., "/media-stream").
func (p *Provider) Listen(ctx context.Context, addr string) (<-chan transport.Connection, error) {
	connCh := make(chan transport.Connection, 10)

	p.mu.Lock()
	p.listeners[addr] = connCh
	p.mu.Unlock()

	return connCh, nil
}

// HandleWebSocket handles an incoming WebSocket connection from Twilio.
// This should be called from your HTTP WebSocket handler.
func (p *Provider) HandleWebSocket(w http.ResponseWriter, r *http.Request, listenerPath string) error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("websocket upgrade failed: %w", err)
	}

	conn := &Connection{
		id:       "", // Will be set from start message
		wsConn:   wsConn,
		provider: p,
		events:   make(chan transport.Event, 100),
		audioIn:  newAudioWriter(),
		audioOut: newAudioReader(),
		done:     make(chan struct{}),
	}

	// Start read/write loops
	go conn.readLoop()
	go conn.writeLoop()

	// Notify listener
	p.mu.RLock()
	listener, ok := p.listeners[listenerPath]
	p.mu.RUnlock()

	if ok {
		select {
		case listener <- conn:
		default:
		}
	}

	return nil
}

// Connect initiates an outbound connection (not typically used for Media Streams).
func (p *Provider) Connect(ctx context.Context, addr string, config transport.Config) (transport.Connection, error) {
	return nil, fmt.Errorf("outbound connections not supported for Media Streams; use CallSystem.MakeCall instead")
}

// Close shuts down the transport.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		_ = conn.Close()
	}

	for _, ch := range p.listeners {
		close(ch)
	}

	p.connections = make(map[string]*Connection)
	p.listeners = make(map[string]chan transport.Connection)

	return nil
}

// SendDTMF sends DTMF tones (not directly supported via Media Streams).
func (p *Provider) SendDTMF(conn transport.Connection, digits string) error {
	return fmt.Errorf("DTMF sending not supported via Media Streams; use TwiML <Play> verb")
}

// OnDTMF sets the DTMF handler.
func (p *Provider) OnDTMF(handler func(conn transport.Connection, digit string)) {
	p.mu.Lock()
	p.dtmfHandler = handler
	p.mu.Unlock()
}

// Transfer transfers the call (requires Twilio API call).
func (p *Provider) Transfer(conn transport.Connection, target string) error {
	return fmt.Errorf("transfer not implemented; use CallSystem to update the call")
}

// Hold places the call on hold.
func (p *Provider) Hold(conn transport.Connection) error {
	return fmt.Errorf("hold not implemented; use CallSystem to update the call")
}

// Unhold resumes a held call.
func (p *Provider) Unhold(conn transport.Connection) error {
	return fmt.Errorf("unhold not implemented; use CallSystem to update the call")
}

// Connection implements transport.Connection for Twilio Media Streams.
type Connection struct {
	id         string
	streamSID  string
	callSID    string
	wsConn     *websocket.Conn
	provider   *Provider
	events     chan transport.Event
	audioIn    *audioWriter
	audioOut   *audioReader
	done       chan struct{}
	mu         sync.RWMutex
	closed     bool
	closeOnce  sync.Once
	remoteAddr net.Addr
}

// ID returns the connection identifier (stream SID).
func (c *Connection) ID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.streamSID
}

// CallSID returns the associated call SID.
func (c *Connection) CallSID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.callSID
}

// AudioIn returns a writer for sending audio to Twilio.
func (c *Connection) AudioIn() io.WriteCloser {
	return c.audioIn
}

// AudioOut returns a reader for receiving audio from Twilio.
func (c *Connection) AudioOut() io.Reader {
	return c.audioOut
}

// Events returns a channel for transport events.
func (c *Connection) Events() <-chan transport.Event {
	return c.events
}

// Close closes the connection.
func (c *Connection) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()

		close(c.done)
		_ = c.audioIn.Close()
		c.audioOut.close()
		close(c.events)
		_ = c.wsConn.Close()

		c.provider.mu.Lock()
		delete(c.provider.connections, c.streamSID)
		c.provider.mu.Unlock()
	})
	return nil
}

// RemoteAddr returns the remote address.
func (c *Connection) RemoteAddr() net.Addr {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.remoteAddr
}

// Twilio Media Streams message types.
type mediaMessage struct {
	Event     string        `json:"event"`
	StreamSID string        `json:"streamSid,omitempty"`
	Start     *startMessage `json:"start,omitempty"`
	Media     *mediaPayload `json:"media,omitempty"`
	Mark      *markMessage  `json:"mark,omitempty"`
	Stop      *stopMessage  `json:"stop,omitempty"`
	DTMF      *dtmfMessage  `json:"dtmf,omitempty"`
}

type startMessage struct {
	StreamSID    string            `json:"streamSid"`
	AccountSID   string            `json:"accountSid"`
	CallSID      string            `json:"callSid"`
	Tracks       []string          `json:"tracks"`
	MediaFormat  mediaFormat       `json:"mediaFormat"`
	CustomParams map[string]string `json:"customParameters"`
}

type mediaFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
}

type mediaPayload struct {
	Track     string `json:"track"`
	Chunk     string `json:"chunk"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"` // Base64 encoded audio
}

type markMessage struct {
	Name string `json:"name"`
}

type stopMessage struct {
	AccountSID string `json:"accountSid"`
	CallSID    string `json:"callSid"`
}

type dtmfMessage struct {
	Digit string `json:"digit"`
}

// readLoop reads messages from the WebSocket.
func (c *Connection) readLoop() {
	defer func() { _ = c.Close() }()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_, data, err := c.wsConn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.events <- transport.Event{Type: transport.EventError, Error: err}
			}
			return
		}

		var msg mediaMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Event {
		case "connected":
			// Connection established
			c.events <- transport.Event{Type: transport.EventConnected}

		case "start":
			if msg.Start != nil {
				c.mu.Lock()
				c.streamSID = msg.Start.StreamSID
				c.callSID = msg.Start.CallSID
				c.mu.Unlock()

				c.provider.mu.Lock()
				c.provider.connections[c.streamSID] = c
				c.provider.mu.Unlock()

				c.events <- transport.Event{Type: transport.EventAudioStarted}
			}

		case "media":
			if msg.Media != nil && msg.Media.Payload != "" {
				// Decode base64 audio
				audio, err := base64.StdEncoding.DecodeString(msg.Media.Payload)
				if err != nil {
					continue
				}
				// Write to audio output
				c.audioOut.write(audio)
			}

		case "dtmf":
			if msg.DTMF != nil {
				c.events <- transport.Event{
					Type: transport.EventDTMF,
					Data: msg.DTMF.Digit,
				}

				c.provider.mu.RLock()
				handler := c.provider.dtmfHandler
				c.provider.mu.RUnlock()

				if handler != nil {
					handler(c, msg.DTMF.Digit)
				}
			}

		case "stop":
			c.events <- transport.Event{Type: transport.EventAudioStopped}
			c.events <- transport.Event{Type: transport.EventDisconnected}
			return

		case "mark":
			// Mark event - can be used for synchronization
		}
	}
}

// writeLoop writes audio to the WebSocket.
func (c *Connection) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case audio := <-c.audioIn.ch:
			// Encode audio to base64
			encoded := base64.StdEncoding.EncodeToString(audio)

			msg := map[string]any{
				"event":     "media",
				"streamSid": c.streamSID,
				"media": map[string]string{
					"payload": encoded,
				},
			}

			c.mu.RLock()
			closed := c.closed
			c.mu.RUnlock()

			if !closed {
				if err := c.wsConn.WriteJSON(msg); err != nil {
					return
				}
			}
		}
	}
}

// SendMark sends a mark message for synchronization.
func (c *Connection) SendMark(name string) error {
	msg := map[string]any{
		"event":     "mark",
		"streamSid": c.streamSID,
		"mark": map[string]string{
			"name": name,
		},
	}
	return c.wsConn.WriteJSON(msg)
}

// Clear clears the audio buffer.
func (c *Connection) Clear() error {
	msg := map[string]any{
		"event":     "clear",
		"streamSid": c.streamSID,
	}
	return c.wsConn.WriteJSON(msg)
}

// audioWriter implements io.WriteCloser for sending audio.
type audioWriter struct {
	ch     chan []byte
	closed bool
	mu     sync.Mutex
}

func newAudioWriter() *audioWriter {
	return &audioWriter{
		ch: make(chan []byte, 100),
	}
}

func (w *audioWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, io.ErrClosedPipe
	}

	// Copy the data
	data := make([]byte, len(p))
	copy(data, p)

	select {
	case w.ch <- data:
		return len(p), nil
	default:
		// Buffer full, drop oldest
		select {
		case <-w.ch:
		default:
		}
		w.ch <- data
		return len(p), nil
	}
}

func (w *audioWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.closed {
		w.closed = true
		close(w.ch)
	}
	return nil
}

// audioReader implements io.Reader for receiving audio.
type audioReader struct {
	ch     chan []byte
	buffer []byte
	mu     sync.Mutex
}

func newAudioReader() *audioReader {
	return &audioReader{
		ch: make(chan []byte, 100),
	}
}

func (r *audioReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If we have buffered data, return it first
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Wait for new data
	data, ok := <-r.ch
	if !ok {
		return 0, io.EOF
	}

	n = copy(p, data)
	if n < len(data) {
		r.buffer = data[n:]
	}
	return n, nil
}

func (r *audioReader) write(data []byte) {
	select {
	case r.ch <- data:
	default:
		// Buffer full, drop
	}
}

func (r *audioReader) close() {
	close(r.ch)
}
