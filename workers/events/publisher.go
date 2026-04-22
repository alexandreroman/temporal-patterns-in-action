package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Publisher sends an event envelope out to subscribers. The category and
// subject are derived from env.Type and env.WorkflowID.
type Publisher interface {
	Publish(ctx context.Context, pattern string, env Envelope) error
	Close() error
}

// NopPublisher is a no-op Publisher used when NATS is unavailable or in tests.
type NopPublisher struct{}

func (NopPublisher) Publish(context.Context, string, Envelope) error { return nil }
func (NopPublisher) Close() error                                    { return nil }

// NATSPublisher publishes events as JSON NATS messages. We issue a short
// FlushTimeout after each publish so SSE-style consumers see events promptly.
type NATSPublisher struct {
	Conn *nats.Conn
}

// Close closes the underlying NATS connection.
func (p *NATSPublisher) Close() error {
	p.Conn.Close()
	return nil
}

// Publish marshals the envelope and publishes it to the derived subject.
func (p *NATSPublisher) Publish(_ context.Context, pattern string, env Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	subject := Subject(pattern, env.WorkflowID, CategoryOf(env.Type))
	if err := p.Conn.Publish(subject, data); err != nil {
		return fmt.Errorf("nats publish %s: %w", subject, err)
	}
	// Flush so demo consumers observe events without waiting for the next
	// protocol write. The bound is low to avoid stalling activity workers.
	if err := p.Conn.FlushTimeout(100 * time.Millisecond); err != nil {
		return fmt.Errorf("nats flush: %w", err)
	}
	return nil
}

// NewPublisher dials NATS and returns a NATSPublisher. When url is empty it
// returns a NopPublisher — useful for local dev without the broker.
func NewPublisher(url string) (Publisher, error) {
	if strings.TrimSpace(url) == "" {
		return NopPublisher{}, nil
	}
	nc, err := nats.Connect(url,
		nats.Name("temporal-patterns-worker"),
		nats.Timeout(5*time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats %s: %w", url, err)
	}
	return &NATSPublisher{Conn: nc}, nil
}
