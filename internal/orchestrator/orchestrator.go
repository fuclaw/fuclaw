package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"fuclaw/internal/db"
	"fuclaw/internal/types"
	"fuclaw/internal/container"
)

type Orchestrator struct {
	db              *db.DB
	channels        []types.Channel
	runner          *container.Runner
	lastTimestamp   string
	registeredGroups map[string]types.RegisteredGroup
	mu              sync.RWMutex

	pollInterval      time.Duration
	schedulerInterval time.Duration
	ipcInterval       time.Duration
	ipcDir            string
}

func NewOrchestrator(db *db.DB, runner *container.Runner, pollInterval time.Duration, schedulerInterval time.Duration, ipcInterval time.Duration, ipcDir string) *Orchestrator {
	return &Orchestrator{
		db:              db,
		runner:          runner,
		registeredGroups: make(map[string]types.RegisteredGroup),
		pollInterval:      pollInterval,
		schedulerInterval: schedulerInterval,
		ipcInterval:       ipcInterval,
		ipcDir:            ipcDir,
	}
}

func (o *Orchestrator) AddChannel(ch types.Channel) {
	o.channels = append(o.channels, ch)
}

func (o *Orchestrator) Start(ctx context.Context) error {
	// Load registered groups
	groups, err := o.db.GetAllRegisteredGroups()
	if err != nil {
		return err
	}
	o.registeredGroups = groups

	// Start channels
	for _, ch := range o.channels {
		if err := ch.Connect(ctx); err != nil {
			log.Printf("Error connecting channel %s: %v", ch.Name(), err)
		}
	}

	// Start scheduler
	go o.startScheduler(ctx)

	// Start IPC watcher
	go o.startIpcWatcher(ctx)

	// Main loop
	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := o.pollMessages(ctx); err != nil {
				log.Printf("Error polling messages: %v", err)
			}
		}
	}
}

func (o *Orchestrator) OnMessage(msg types.NewMessage) {
	if err := o.db.StoreMessage(msg); err != nil {
		log.Printf("Error storing message: %v", err)
	}
}

func (o *Orchestrator) PollOnce(ctx context.Context) error {
	return o.pollMessages(ctx)
}

func (o *Orchestrator) SetRegisteredGroupForTest(group types.RegisteredGroup) {
	o.registeredGroups[group.JID] = group
}

func (o *Orchestrator) pollMessages(ctx context.Context) error {
	jids := make([]string, 0, len(o.registeredGroups))
	for jid := range o.registeredGroups {
		jids = append(jids, jid)
	}

	messages, newTs, err := o.db.GetNewMessages(jids, o.lastTimestamp)
	if err != nil {
		return err
	}

	if len(messages) > 0 {
		o.lastTimestamp = newTs
		// Group by chat
		byChat := make(map[string][]types.NewMessage)
		for _, m := range messages {
			byChat[m.ChatJID] = append(byChat[m.ChatJID], m)
		}

		for jid, msgs := range byChat {
			group, ok := o.registeredGroups[jid]
			if !ok {
				continue
			}
			if err := o.processGroup(ctx, group, msgs); err != nil {
				log.Printf("Error running agent for group %s: %v", group.Name, err)
			}
		}
	}

	return nil
}

func (o *Orchestrator) processGroup(ctx context.Context, group types.RegisteredGroup, msgs []types.NewMessage) error {
	if o.runner == nil {
		return fmt.Errorf("container runner not initialized")
	}

	// Build prompt from messages
	prompt := ""
	for _, m := range msgs {
		prompt += fmt.Sprintf("[%s] %s: %s\n", m.Timestamp.Format("Jan 02 3:04 PM"), m.SenderName, m.Content)
	}

	input := types.ContainerInput{
		Prompt:        prompt,
		GroupFolder:   group.Folder,
		ChatJID:       group.JID,
		IsMain:        group.IsMain,
		AssistantName: "Fuclaw",
	}

	onOutput := func(output types.ContainerOutput) {
		if output.Result != nil {
			text := fmt.Sprintf("%v", output.Result)
			// Find owning channel
			for _, ch := range o.channels {
				if ch.OwnsJid(group.JID) {
					ch.SendMessage(ctx, group.JID, text)
					break
				}
			}
		}
	}

	if err := o.runner.RunAgent(ctx, group, input, onOutput); err != nil {
		return err
	}
	return nil
}
