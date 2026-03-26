package db

import (
	"path/filepath"
	"testing"
	"time"

	"fuclaw/internal/types"
)

func TestInitDBAndMessageRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "messages.db")

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB error: %v", err)
	}
	defer database.conn.Close()

	group := types.RegisteredGroup{
		JID:             "test_jid",
		Name:            "Test Group",
		Folder:          "test_folder",
		Trigger:         "",
		AddedAt:         time.Now(),
		RequiresTrigger: false,
		IsMain:          true,
	}
	if err := database.SetRegisteredGroup(group); err != nil {
		t.Fatalf("SetRegisteredGroup error: %v", err)
	}

	msg := types.NewMessage{
		ID:         "m1",
		ChatJID:    group.JID,
		Sender:     "me",
		SenderName: "Me",
		Content:    "hello",
		Timestamp:  time.Now(),
		IsFromMe:   false,
		IsBot:      false,
	}
	if err := database.StoreMessage(msg); err != nil {
		t.Fatalf("StoreMessage error: %v", err)
	}

	got, newTs, err := database.GetNewMessages([]string{group.JID}, "")
	if err != nil {
		t.Fatalf("GetNewMessages error: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected at least 1 message")
	}
	if newTs == "" {
		t.Fatalf("expected newTs not empty")
	}
}

func TestScheduledTaskQueries(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "messages.db")

	database, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB error: %v", err)
	}
	defer database.conn.Close()

	nextRun := time.Now().Add(-1 * time.Minute).Format(time.RFC3339)
	createdAt := time.Now().Format(time.RFC3339)
	_, err = database.conn.Exec(
		`INSERT INTO scheduled_tasks (id, group_folder, chat_jid, prompt, schedule_type, schedule_value, next_run, created_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"t1", "test_folder", "test_jid", "do something", "cron", "* * * * *", nextRun, createdAt, "active",
	)
	if err != nil {
		t.Fatalf("insert scheduled_tasks error: %v", err)
	}

	tasks, err := database.GetDueTasks()
	if err != nil {
		t.Fatalf("GetDueTasks error: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatalf("expected due tasks")
	}

	if err := database.UpdateTaskAfterRun("t1", time.Now().Add(1*time.Minute), "ok"); err != nil {
		t.Fatalf("UpdateTaskAfterRun error: %v", err)
	}

	if err := database.LogTaskRun(types.TaskRunLog{
		TaskID:     "t1",
		RunAt:      time.Now(),
		DurationMS: 10,
		Status:     "success",
		Result:     "ok",
		Error:      "",
	}); err != nil {
		t.Fatalf("LogTaskRun error: %v", err)
	}
}

