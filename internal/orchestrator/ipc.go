package orchestrator

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fuclaw/internal/types"
)

func (o *Orchestrator) ProcessIpcOnce(ctx context.Context) {
	ipcDir := o.ipcDir
	if ipcDir == "" {
		cwd, _ := os.Getwd()
		ipcDir = filepath.Join(cwd, "data", "ipc")
	}
	_ = os.MkdirAll(ipcDir, 0755)
	o.checkIpc(ctx, ipcDir)
}

func (o *Orchestrator) startIpcWatcher(ctx context.Context) {
	ipcDir := o.ipcDir
	if ipcDir == "" {
		cwd, _ := os.Getwd()
		ipcDir = filepath.Join(cwd, "data", "ipc")
	}
	os.MkdirAll(ipcDir, 0755)

	ticker := time.NewTicker(o.ipcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.checkIpc(ctx, ipcDir)
		}
	}
}

func (o *Orchestrator) checkIpc(ctx context.Context, ipcDir string) {
	o.mu.RLock()
	groups := make([]types.RegisteredGroup, 0, len(o.registeredGroups))
	for _, g := range o.registeredGroups {
		groups = append(groups, g)
	}
	o.mu.RUnlock()

	for _, g := range groups {
		groupIpcDir := filepath.Join(ipcDir, g.Folder)

		// 1. Check for outbound messages
		msgDir := filepath.Join(groupIpcDir, "messages")
		o.processIpcMessages(ctx, g.JID, msgDir)

		// 2. Check for task operations
		taskDir := filepath.Join(groupIpcDir, "tasks")
		o.processIpcTasks(ctx, g, taskDir)
	}
}

func (o *Orchestrator) processIpcMessages(ctx context.Context, jid string, dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "msg-") {
			continue
		}

		path := filepath.Join(dir, f.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		var msg struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(data, &msg); err == nil {
			for _, ch := range o.channels {
				if ch.OwnsJid(jid) {
					ch.SendMessage(ctx, jid, msg.Text)
					break
				}
			}
		}

		os.Remove(path)
	}
}

func (o *Orchestrator) processIpcTasks(ctx context.Context, group types.RegisteredGroup, dir string) {
	// Simple implementation: only handle task creation for now
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "task-") {
			continue
		}

		path := filepath.Join(dir, f.Name())
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		log.Printf("IPC Task op received: %s", string(data))

		os.Remove(path)
	}
}
