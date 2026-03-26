package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"fuclaw/internal/types"
)

type DB struct {
	conn *sql.DB
}

func InitDB(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.createSchema(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS chats (
		jid TEXT PRIMARY KEY,
		name TEXT,
		last_message_time TEXT,
		channel TEXT,
		is_group INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS messages (
		id TEXT,
		chat_jid TEXT,
		sender TEXT,
		sender_name TEXT,
		content TEXT,
		timestamp TEXT,
		is_from_me INTEGER,
		is_bot_message INTEGER DEFAULT 0,
		PRIMARY KEY (id, chat_jid),
		FOREIGN KEY (chat_jid) REFERENCES chats(jid)
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON messages(timestamp);

	CREATE TABLE IF NOT EXISTS registered_groups (
		jid TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		folder TEXT NOT NULL UNIQUE,
		trigger_pattern TEXT NOT NULL,
		added_at TEXT NOT NULL,
		container_config TEXT,
		requires_trigger INTEGER DEFAULT 1,
		is_main INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS scheduled_tasks (
		id TEXT PRIMARY KEY,
		group_folder TEXT NOT NULL,
		chat_jid TEXT NOT NULL,
		prompt TEXT NOT NULL,
		schedule_type TEXT NOT NULL,
		schedule_value TEXT NOT NULL,
		next_run TEXT,
		last_run TEXT,
		last_result TEXT,
		status TEXT DEFAULT 'active',
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_next_run ON scheduled_tasks(next_run);
	CREATE INDEX IF NOT EXISTS idx_status ON scheduled_tasks(status);

	CREATE TABLE IF NOT EXISTS task_run_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		run_at TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		status TEXT NOT NULL,
		result TEXT,
		error TEXT,
		FOREIGN KEY (task_id) REFERENCES scheduled_tasks(id)
	);
	CREATE INDEX IF NOT EXISTS idx_task_run_logs ON task_run_logs(task_id, run_at);
	`
	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) StoreMessage(msg types.NewMessage) error {
	query := `INSERT OR REPLACE INTO messages (id, chat_jid, sender, sender_name, content, timestamp, is_from_me, is_bot_message) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, msg.ID, msg.ChatJID, msg.Sender, msg.SenderName, msg.Content, msg.Timestamp.Format(time.RFC3339), boolToInt(msg.IsFromMe), boolToInt(msg.IsBot))
	return err
}

func (db *DB) GetNewMessages(jids []string, lastTimestamp string) ([]types.NewMessage, string, error) {
	if len(jids) == 0 {
		return nil, lastTimestamp, nil
	}

	// Simple implementation for now
	query := `SELECT id, chat_jid, sender, sender_name, content, timestamp, is_from_me, is_bot_message FROM messages WHERE timestamp > ? ORDER BY timestamp`
	rows, err := db.conn.Query(query, lastTimestamp)
	if err != nil {
		return nil, lastTimestamp, err
	}
	defer rows.Close()

	var messages []types.NewMessage
	newTimestamp := lastTimestamp

	for rows.Next() {
		var m types.NewMessage
		var ts string
		var fromMe, isBot int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.Sender, &m.SenderName, &m.Content, &ts, &fromMe, &isBot); err != nil {
			return nil, lastTimestamp, err
		}
		m.Timestamp, _ = time.Parse(time.RFC3339, ts)
		m.IsFromMe = intToBool(fromMe)
		m.IsBot = intToBool(isBot)
		messages = append(messages, m)
		if ts > newTimestamp {
			newTimestamp = ts
		}
	}

	return messages, newTimestamp, nil
}

func (db *DB) GetAllRegisteredGroups() (map[string]types.RegisteredGroup, error) {
	rows, err := db.conn.Query("SELECT jid, name, folder, trigger_pattern, added_at, container_config, requires_trigger, is_main FROM registered_groups")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make(map[string]types.RegisteredGroup)
	for rows.Next() {
		var g types.RegisteredGroup
		var addedAt, config string
		var reqTrigger, isMain int
		if err := rows.Scan(&g.JID, &g.Name, &g.Folder, &g.Trigger, &addedAt, &config, &reqTrigger, &isMain); err != nil {
			return nil, err
		}
		g.AddedAt, _ = time.Parse(time.RFC3339, addedAt)
		g.ContainerConfig = config
		g.RequiresTrigger = intToBool(reqTrigger)
		g.IsMain = intToBool(isMain)
		groups[g.JID] = g
	}
	return groups, nil
}

func (db *DB) SetRegisteredGroup(g types.RegisteredGroup) error {
	query := `INSERT OR REPLACE INTO registered_groups (jid, name, folder, trigger_pattern, added_at, container_config, requires_trigger, is_main) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, g.JID, g.Name, g.Folder, g.Trigger, g.AddedAt.Format(time.RFC3339), g.ContainerConfig, boolToInt(g.RequiresTrigger), boolToInt(g.IsMain))
	return err
}

func (db *DB) GetDueTasks() ([]types.ScheduledTask, error) {
	now := time.Now().Format(time.RFC3339)
	query := `SELECT id, group_folder, chat_jid, prompt, schedule_type, schedule_value, next_run, status FROM scheduled_tasks WHERE status = 'active' AND next_run <= ?`
	rows, err := db.conn.Query(query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []types.ScheduledTask
	for rows.Next() {
		var t types.ScheduledTask
		var nextRun string
		if err := rows.Scan(&t.ID, &t.GroupFolder, &t.ChatJID, &t.Prompt, &t.ScheduleType, &t.ScheduleValue, &nextRun, &t.Status); err != nil {
			return nil, err
		}
		t.NextRun, _ = time.Parse(time.RFC3339, nextRun)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (db *DB) UpdateTaskAfterRun(id string, nextRun time.Time, lastResult string) error {
	query := `UPDATE scheduled_tasks SET next_run = ?, last_run = ?, last_result = ? WHERE id = ?`
	now := time.Now().Format(time.RFC3339)
	_, err := db.conn.Exec(query, nextRun.Format(time.RFC3339), now, lastResult, id)
	return err
}

func (db *DB) LogTaskRun(log types.TaskRunLog) error {
	query := `INSERT INTO task_run_logs (task_id, run_at, duration_ms, status, result, error) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, log.TaskID, log.RunAt.Format(time.RFC3339), log.DurationMS, log.Status, log.Result, log.Error)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i == 1
}
