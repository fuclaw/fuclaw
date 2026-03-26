package testutil

import (
	"context"
	"sync"

	"fuclaw/internal/types"
)

type SentMessage struct {
	JID  string
	Text string
}

type TestChannel struct {
	name      string
	owns      func(string) bool
	onMessage func(types.NewMessage)

	mu        sync.RWMutex
	connected bool
	sent      chan SentMessage
}

func NewTestChannel(name string, owns func(string) bool, onMessage func(types.NewMessage)) *TestChannel {
	if name == "" {
		name = "test"
	}
	if owns == nil {
		owns = func(string) bool { return true }
	}
	return &TestChannel{
		name:      name,
		owns:      owns,
		onMessage: onMessage,
		sent:      make(chan SentMessage, 100),
	}
}

func (c *TestChannel) Name() string {
	return c.name
}

func (c *TestChannel) Connect(ctx context.Context) error {
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	return nil
}

func (c *TestChannel) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
	return nil
}

func (c *TestChannel) SendMessage(ctx context.Context, jid string, text string) error {
	c.sent <- SentMessage{JID: jid, Text: text}
	return nil
}

func (c *TestChannel) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *TestChannel) OwnsJid(jid string) bool {
	return c.owns(jid)
}

func (c *TestChannel) Inject(msg types.NewMessage) {
	if c.onMessage != nil {
		c.onMessage(msg)
	}
}

func (c *TestChannel) Sent() <-chan SentMessage {
	return c.sent
}

