package channel

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"fuclaw/internal/types"
)

// ConsoleChannel is a terminal-based channel for interaction
type ConsoleChannel struct {
	onMessage func(types.NewMessage)
	connected bool
	cancel    context.CancelFunc
}

func NewConsoleChannel(onMessage func(types.NewMessage)) *ConsoleChannel {
	return &ConsoleChannel{
		onMessage: onMessage,
	}
}

func (c *ConsoleChannel) Name() string {
	return "console"
}

func (c *ConsoleChannel) Connect(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	c.connected = true
	go c.readLoop(ctx)
	fmt.Println("Console channel connected. Type your message:")
	return nil
}

func (c *ConsoleChannel) Disconnect(ctx context.Context) error {
	c.connected = false
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func (c *ConsoleChannel) SendMessage(ctx context.Context, jid string, text string) error {
	fmt.Printf("\n[Fuclaw] -> %s: %s\n", jid, text)
	return nil
}

func (c *ConsoleChannel) IsConnected() bool {
	return c.connected
}

func (c *ConsoleChannel) OwnsJid(jid string) bool {
	return jid == "console_user"
}

func (c *ConsoleChannel) readLoop(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Print("> ")
			input, err := reader.ReadString('\n')
			if err != nil {
				if ctx.Err() == nil {
					fmt.Printf("Error reading console: %v\n", err)
				}
				return
			}
			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}

			msg := types.NewMessage{
				ID:         fmt.Sprintf("console-%d", time.Now().UnixNano()),
				ChatJID:    "console_user",
				Sender:     "me",
				SenderName: "Console User",
				Content:    input,
				Timestamp:  time.Now(),
				IsFromMe:   false,
				IsBot:      false,
			}
			c.onMessage(msg)
		}
	}
}
