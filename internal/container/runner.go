package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	fctypes "fuclaw/internal/types"
)

const (
	OutputStartMarker = "---FUCLAW_OUTPUT_START---"
	OutputEndMarker   = "---FUCLAW_OUTPUT_END---"
)

type Runner struct {
	docker *client.Client
	image  string
	groupsDir string
	ipcDir    string
}

func NewRunner(image string, groupsDir string, ipcDir string) (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Runner{
		docker: cli,
		image:  image,
		groupsDir: groupsDir,
		ipcDir:    ipcDir,
	}, nil
}

func (r *Runner) RunAgent(ctx context.Context, group fctypes.RegisteredGroup, input fctypes.ContainerInput, onOutput func(fctypes.ContainerOutput)) error {
	containerName := fmt.Sprintf("fuclaw-%s-%d", group.Folder, time.Now().UnixNano())

	groupsDir := r.groupsDir
	if groupsDir == "" {
		cwd, _ := os.Getwd()
		groupsDir = filepath.Join(cwd, "groups")
	}
	ipcDir := r.ipcDir
	if ipcDir == "" {
		cwd, _ := os.Getwd()
		ipcDir = filepath.Join(cwd, "data", "ipc")
	}
	_ = os.MkdirAll(filepath.Join(groupsDir, group.Folder), 0755)
	_ = os.MkdirAll(filepath.Join(ipcDir, group.Folder), 0755)

	config := &container.Config{
		Image:        r.image,
		Tty:          false,
		OpenStdin:    true,
		StdinOnce:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env: buildContainerEnv(),
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(groupsDir, group.Folder),
				Target: "/workspace/group",
			},
			{
				Type:   mount.TypeBind,
				Source: filepath.Join(ipcDir, group.Folder),
				Target: "/workspace/ipc",
			},
		},
	}

	resp, err := r.docker.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return err
	}

	hijack, err := r.docker.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return err
	}
	defer hijack.Close()

	if err := r.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	// Write input to container stdin
	inputJSON, _ := json.Marshal(input)
	if _, err := hijack.Conn.Write(append(inputJSON, '\n')); err != nil {
		return err
	}

	// Read output from container stdout
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		scanner := bufio.NewScanner(hijack.Reader)
		for scanner.Scan() {
			line := scanner.Text()
			output, matched, err := ParseOutputLine(line)
			if err != nil {
				log.Printf("[%s] output parse error: %v", group.Name, err)
				continue
			}
			if matched && output != nil {
				onOutput(*output)
				continue
			}
			log.Printf("[%s] %s", group.Name, line)
		}
	}()

	// Wait for container to finish
	statusCh, errCh := r.docker.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil && err != io.EOF {
			return err
		}
	case <-statusCh:
	case <-ctx.Done():
		r.docker.ContainerStop(ctx, resp.ID, container.StopOptions{})
		return ctx.Err()
	}

	select {
	case <-readDone:
	case <-time.After(2 * time.Second):
	}

	return nil
}

func buildContainerEnv() []string {
	keys := []string{
		"FUCLAW_OPENAI_BASE_URL",
		"FUCLAW_OPENAI_MODEL",
		"FUCLAW_OPENAI_API_KEY",
	}
	env := make([]string, 0, len(keys))
	for _, k := range keys {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" {
			continue
		}
		env = append(env, k+"="+v)
	}
	return env
}
