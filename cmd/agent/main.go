package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"fuclaw/internal/config"
	"fuclaw/internal/types"

	"github.com/joho/godotenv"
)

const (
	OutputStartMarker = "---FUCLAW_OUTPUT_START---"
	OutputEndMarker   = "---FUCLAW_OUTPUT_END---"
)

// MySendMessageTool implements tool.InvokableTool
type MySendMessageTool struct{}

func (t *MySendMessageTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "send_message",
		Desc: "Send a message back to the group chat",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"text": {
				Type:     schema.String,
				Desc:     "The message text to send",
				Required: true,
			},
		}),
	}, nil
}

func (t *MySendMessageTool) InvokableRun(ctx context.Context, inputStr string, opts ...tool.Option) (string, error) {
	// Write to IPC messages directory
	ipcDir := "/workspace/ipc/messages"
	os.MkdirAll(ipcDir, 0755)
	filename := fmt.Sprintf("msg-%d.json", time.Now().UnixNano())
	os.WriteFile(filepath.Join(ipcDir, filename), []byte(inputStr), 0644)
	return "Message sent successfully", nil
}

func main() {
	ctx := context.Background()

	// Read input from stdin
	reader := bufio.NewReader(os.Stdin)
	inputBytes, err := reader.ReadBytes('\n')
	if err != nil {
		sendError(err.Error())
		return
	}

	var input types.ContainerInput
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		sendError("Failed to unmarshal input: " + err.Error())
		return
	}

	_ = godotenv.Load()
	cfg, _ := config.Load("")
	if err := config.ValidateOpenAI(cfg); err != nil {
		sendError(err.Error())
		return
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.OpenAIBaseURL,
		Model:   cfg.OpenAIModel,
		APIKey:  cfg.OpenAIApiKey,
	})
	if err != nil {
		sendError("Failed to create chat model: " + err.Error())
		return
	}

	// Create ADK Agent
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        input.AssistantName,
		Instruction: fmt.Sprintf("You are %s, a helpful assistant.", input.AssistantName),
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{&MySendMessageTool{}},
			},
		},
	})
	if err != nil {
		sendError("Failed to create agent: " + err.Error())
		return
	}

	// Create Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})

	// Run query
	iter := runner.Query(ctx, input.Prompt)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event != nil && event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput
			if !mv.IsStreaming && mv.Message != nil && mv.Message.Content != "" {
				sendOutput(types.ContainerOutput{
					Status: "success",
					Result: mv.Message.Content,
				})
			} else if mv.IsStreaming && mv.MessageStream != nil {
				// Handle stream
				for {
					msg, err := mv.MessageStream.Recv()
					if err != nil {
						break
					}
					sendOutput(types.ContainerOutput{
						Status: "success",
						Result: msg.Content,
					})
				}
			}
		}
	}

	// Final success message
	sendOutput(types.ContainerOutput{
		Status: "success",
		Result: nil,
	})
}

func sendOutput(out types.ContainerOutput) {
	bytes, _ := json.Marshal(out)
	fmt.Printf("%s%s%s\n", OutputStartMarker, string(bytes), OutputEndMarker)
}

func sendError(errStr string) {
	out := types.ContainerOutput{
		Status: "error",
		Error:  errStr,
	}
	bytes, _ := json.Marshal(out)
	fmt.Printf("%s%s%s\n", OutputStartMarker, string(bytes), OutputEndMarker)
}
