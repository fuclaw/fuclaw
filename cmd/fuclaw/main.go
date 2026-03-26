package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"fuclaw/internal/channel"
	"fuclaw/internal/config"
	"fuclaw/internal/container"
	"fuclaw/internal/db"
	"fuclaw/internal/orchestrator"
	"fuclaw/internal/types"

	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cwd, _ := os.Getwd()

	_ = godotenv.Load()
	cfg, err := config.Load(cwd)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	runner, err := container.NewRunner(cfg.DockerImage, cfg.GroupsDir, cfg.IpcDir)
	if err != nil {
		log.Printf("Warning: Docker runner failed to initialize: %v", err)
	}

	orch := orchestrator.NewOrchestrator(database, runner, cfg.PollInterval, cfg.SchedulerInterval, cfg.IpcInterval, cfg.IpcDir)

	console := channel.NewConsoleChannel(orch.OnMessage)
	orch.AddChannel(console)

	groups, _ := database.GetAllRegisteredGroups()
	if _, ok := groups["console_user"]; !ok {
		testGroup := types.RegisteredGroup{
			JID:             "console_user",
			Name:            "Console Group",
			Folder:          "console",
			Trigger:         "", // No trigger for console
			AddedAt:         time.Now(),
			RequiresTrigger: false,
			IsMain:          true,
		}
		if err := database.SetRegisteredGroup(testGroup); err != nil {
			log.Fatalf("Failed to register test group: %v", err)
		}

		groupDir := filepath.Join(cfg.GroupsDir, "console")
		os.MkdirAll(groupDir, 0755)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	log.Println("Fuclaw starting...")
	if err := orch.Start(ctx); err != nil {
		log.Fatalf("Orchestrator error: %v", err)
	}
}
