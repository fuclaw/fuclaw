package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	DBPath            string
	GroupsDir         string
	IpcDir            string
	DockerImage       string
	PollInterval      time.Duration
	SchedulerInterval time.Duration
	IpcInterval       time.Duration

	OpenAIBaseURL string
	OpenAIModel   string
	OpenAIApiKey  string
}

func Load(cwd string) (Config, error) {
	cfg := Config{
		DBPath:            envOrDefault("FUCLAW_DB_PATH", "./messages.db"),
		GroupsDir:         envOrDefault("FUCLAW_GROUPS_DIR", "./groups"),
		IpcDir:            envOrDefault("FUCLAW_IPC_DIR", "./data/ipc"),
		DockerImage:       envOrDefault("FUCLAW_DOCKER_IMAGE", "fuclaw-agent:latest"),
		PollInterval:      envDurationOrDefault("FUCLAW_POLL_INTERVAL", 2*time.Second),
		SchedulerInterval: envDurationOrDefault("FUCLAW_SCHEDULER_INTERVAL", 1*time.Minute),
		IpcInterval:       envDurationOrDefault("FUCLAW_IPC_INTERVAL", 1*time.Second),
		OpenAIBaseURL:     strings.TrimSpace(os.Getenv("FUCLAW_OPENAI_BASE_URL")),
		OpenAIModel:       strings.TrimSpace(os.Getenv("FUCLAW_OPENAI_MODEL")),
		OpenAIApiKey:      strings.TrimSpace(os.Getenv("FUCLAW_OPENAI_API_KEY")),
	}

	if cwd != "" {
		cfg.DBPath = absFromCwd(cwd, cfg.DBPath)
		cfg.GroupsDir = absFromCwd(cwd, cfg.GroupsDir)
		cfg.IpcDir = absFromCwd(cwd, cfg.IpcDir)
	}

	return cfg, nil
}

func ValidateOpenAI(cfg Config) error {
	var missing []string
	if cfg.OpenAIBaseURL == "" {
		missing = append(missing, "FUCLAW_OPENAI_BASE_URL")
	}
	if cfg.OpenAIModel == "" {
		missing = append(missing, "FUCLAW_OPENAI_MODEL")
	}
	if cfg.OpenAIApiKey == "" {
		missing = append(missing, "FUCLAW_OPENAI_API_KEY")
	}
	if len(missing) > 0 {
		return errors.New("missing required env vars: " + strings.Join(missing, ", "))
	}
	return nil
}

func absFromCwd(cwd string, p string) string {
	if p == "" {
		return p
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(cwd, p)
}

func envOrDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func envDurationOrDefault(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
