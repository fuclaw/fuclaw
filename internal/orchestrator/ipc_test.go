package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fuclaw/internal/testutil"
	"fuclaw/internal/types"
)

func TestProcessIpcOnce_SendsMessageAndDeletesFile(t *testing.T) {
	tmp := t.TempDir()
	ipcDir := filepath.Join(tmp, "ipc")

	o := NewOrchestrator(nil, nil, time.Second, time.Minute, time.Second, ipcDir)

	jid := "test_jid"
	group := types.RegisteredGroup{
		JID:    jid,
		Name:   "Test Group",
		Folder: "g1",
	}
	o.registeredGroups[jid] = group

	ch := testutil.NewTestChannel("test", func(s string) bool { return s == jid }, nil)
	o.AddChannel(ch)

	msgDir := filepath.Join(ipcDir, group.Folder, "messages")
	if err := os.MkdirAll(msgDir, 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	msgPath := filepath.Join(msgDir, "msg-1.json")
	if err := os.WriteFile(msgPath, []byte(`{"text":"hello"}`), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	o.ProcessIpcOnce(context.Background())

	select {
	case sent := <-ch.Sent():
		if sent.JID != jid {
			t.Fatalf("unexpected jid: %s", sent.JID)
		}
		if sent.Text != "hello" {
			t.Fatalf("unexpected text: %s", sent.Text)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for sent message")
	}

	if _, err := os.Stat(msgPath); err == nil {
		t.Fatalf("expected ipc file to be deleted")
	}
}

