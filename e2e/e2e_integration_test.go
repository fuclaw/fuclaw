//go:build integration

package e2e

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"fuclaw/internal/container"
	"fuclaw/internal/db"
	"fuclaw/internal/orchestrator"
	"fuclaw/internal/testutil"
	"fuclaw/internal/types"

	dockerclient "github.com/docker/docker/client"
)

func TestE2E_DockerAgent_WithMockOpenAI(t *testing.T) {
	ctx := context.Background()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}

		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		stream := bytes.Contains(body, []byte(`"stream":true`)) || bytes.Contains(body, []byte(`"stream": true`))
		if stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok-from-mock\"},\"finish_reason\":null}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"id\":\"1\",\"object\":\"chat.completion\",\"choices\":[{\"index\":0,\"message\":{\"role\":\"assistant\",\"content\":\"ok-from-mock\"},\"finish_reason\":\"stop\"}]}"))
	})

	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	mock := httptest.NewUnstartedServer(handler)
	mock.Listener = l
	mock.Start()
	defer mock.Close()

	u, err := url.Parse(mock.URL)
	if err != nil {
		t.Fatalf("parse mock url error: %v", err)
	}
	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split host port error: %v", err)
	}

	imageName := "fuclaw-agent:latest"
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("docker not available: %v", err)
	}
	defer cli.Close()

	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		t.Skipf("docker image not found (%s). build it with: make build-agent", imageName)
	}

	hostForContainer := strings.TrimSpace(os.Getenv("FUCLAW_E2E_HOST"))
	if hostForContainer == "" {
		if runtime.GOOS == "linux" {
			hostForContainer = "172.17.0.1"
		} else {
			hostForContainer = "host.docker.internal"
		}
	}
	t.Setenv("FUCLAW_OPENAI_BASE_URL", "http://"+hostForContainer+":"+port+"/v1")
	t.Setenv("FUCLAW_OPENAI_MODEL", "gpt-4o-mini")
	t.Setenv("FUCLAW_OPENAI_API_KEY", "test")

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "messages.db")
	groupsDir := filepath.Join(tmp, "groups")
	ipcDir := filepath.Join(tmp, "ipc")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB error: %v", err)
	}
	defer database.Close()

	runner, err := container.NewRunner(imageName, groupsDir, ipcDir)
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}

	o := orchestrator.NewOrchestrator(database, runner, 100*time.Millisecond, time.Minute, 100*time.Millisecond, ipcDir)

	jid := "test_jid"
	group := types.RegisteredGroup{
		JID:             jid,
		Name:            "Test Group",
		Folder:          "g1",
		Trigger:         "",
		AddedAt:         time.Now(),
		RequiresTrigger: false,
		IsMain:          true,
	}
	o.SetRegisteredGroupForTest(group)

	ch := testutil.NewTestChannel("test", func(s string) bool { return s == jid }, o.OnMessage)
	o.AddChannel(ch)

	ch.Inject(types.NewMessage{
		ID:         "m1",
		ChatJID:    jid,
		Sender:     "me",
		SenderName: "Me",
		Content:    "ping",
		Timestamp:  time.Now(),
	})

	if err := o.PollOnce(ctx); err != nil {
		t.Fatalf("PollOnce error: %v", err)
	}

	deadline := time.After(30 * time.Second)
	for {
		select {
		case sent := <-ch.Sent():
			if strings.Contains(sent.Text, "ok-from-mock") {
				return
			}
		case <-deadline:
			t.Fatalf("timeout waiting for ok-from-mock response")
		}
	}
}
