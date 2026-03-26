package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"

	"fuclaw/internal/types"
)

func (o *Orchestrator) startScheduler(ctx context.Context) {
	ticker := time.NewTicker(o.schedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tasks, err := o.db.GetDueTasks()
			if err != nil {
				log.Printf("Error getting due tasks: %v", err)
				continue
			}

			for _, t := range tasks {
				go o.runTask(ctx, t)
			}
		}
	}
}

func (o *Orchestrator) runTask(ctx context.Context, task types.ScheduledTask) {
	log.Printf("Running scheduled task %s for group %s", task.ID, task.GroupFolder)
	start := time.Now()

	group, ok := o.registeredGroups[task.ChatJID]
	if !ok {
		log.Printf("Task %s: Group %s not found", task.ID, task.ChatJID)
		return
	}

	input := types.ContainerInput{
		Prompt:        task.Prompt,
		GroupFolder:   task.GroupFolder,
		ChatJID:       task.ChatJID,
		IsMain:        group.IsMain,
		AssistantName: "Fuclaw",
	}

	var lastResult string
	onOutput := func(output types.ContainerOutput) {
		if output.Result != nil {
			lastResult = fmt.Sprintf("%v", output.Result)
			// Send to channel
			for _, ch := range o.channels {
				if ch.OwnsJid(task.ChatJID) {
					ch.SendMessage(ctx, task.ChatJID, lastResult)
					break
				}
			}
		}
	}

	err := o.runner.RunAgent(ctx, group, input, onOutput)
	
	status := "success"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}

	// Calculate next run
	nextRun := o.calculateNextRun(task)

	// Update DB
	o.db.UpdateTaskAfterRun(task.ID, nextRun, lastResult)
	o.db.LogTaskRun(types.TaskRunLog{
		TaskID:     task.ID,
		RunAt:      start,
		DurationMS: time.Since(start).Milliseconds(),
		Status:     status,
		Result:     lastResult,
		Error:      errMsg,
	})
}

func (o *Orchestrator) calculateNextRun(task types.ScheduledTask) time.Time {
	switch task.ScheduleType {
	case "cron":
		schedule, err := cron.ParseStandard(task.ScheduleValue)
		if err != nil {
			return time.Time{}
		}
		return schedule.Next(time.Now())
	case "interval":
		ms, _ := strconv.ParseInt(task.ScheduleValue, 10, 64)
		return time.Now().Add(time.Duration(ms) * time.Millisecond)
	case "once":
		return time.Time{} // Completed
	default:
		return time.Time{}
	}
}
