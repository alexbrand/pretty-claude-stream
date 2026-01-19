package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Event struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event,omitempty"`
}

type AssistantEnvelope struct {
	Type    string            `json:"type"`
	Message *AssistantMessage `json:"message,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type AssistantMessage struct {
	ID      string             `json:"id,omitempty"`
	Role    string             `json:"role,omitempty"`
	Content []AssistantContent `json:"content,omitempty"`
}

type AssistantContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ResultEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
	IsError bool   `json:"is_error,omitempty"`
	Result  string `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

type StreamEvent struct {
	Type         string        `json:"type"`
	Index        int           `json:"index"`
	Delta        *Delta        `json:"delta,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
}

type Delta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type ContentBlock struct {
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Text  string `json:"text,omitempty"`
	Input any    `json:"input,omitempty"`
}

var toolInputs = make(map[int]*strings.Builder)
var lastAssistantText string

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Cyan      = "\033[36m"
	Yellow    = "\033[33m"
	Green     = "\033[32m"
	Blue      = "\033[34m"
	Magenta   = "\033[35m"
	Red       = "\033[31m"
	BoldCyan  = "\033[1;36m"
	BoldGreen = "\033[1;32m"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &base); err != nil {
			continue
		}

		switch base.Type {
		case "stream_event":
			var event Event
			if err := json.Unmarshal(line, &event); err != nil {
				continue
			}
			handleStreamEvent(event.Event)
		case "assistant":
			handleAssistant(line)
		case "result":
			handleResult(line)
		}
	}

	fmt.Println()
}

func handleStreamEvent(raw json.RawMessage) {
	var se StreamEvent
	if err := json.Unmarshal(raw, &se); err != nil {
		return
	}

	switch se.Type {
	case "content_block_start":
		if se.ContentBlock != nil && se.ContentBlock.Type == "tool_use" {
			fmt.Printf("\n%s[Tool: %s%s%s]%s\n", Dim, BoldCyan, se.ContentBlock.Name, Dim, Reset)
			toolInputs[se.Index] = &strings.Builder{}
		}
	case "content_block_delta":
		if se.Delta != nil {
			switch se.Delta.Type {
			case "text_delta":
				fmt.Print(se.Delta.Text)
			case "input_json_delta":
				if sb, ok := toolInputs[se.Index]; ok {
					sb.WriteString(se.Delta.PartialJSON)
				}
			}
		}
	case "content_block_stop":
		if sb, ok := toolInputs[se.Index]; ok {
			printPrettyParams(sb.String())
			delete(toolInputs, se.Index)
		}
	}
}

func handleAssistant(line []byte) {
	var env AssistantEnvelope
	if err := json.Unmarshal(line, &env); err != nil {
		return
	}

	if env.Message == nil {
		if env.Error != "" {
			fmt.Printf("%s%s%s\n", Red, env.Error, Reset)
			lastAssistantText = env.Error
		}
		return
	}

	var sb strings.Builder
	for _, item := range env.Message.Content {
		if item.Type == "text" {
			sb.WriteString(item.Text)
		}
	}
	if sb.Len() > 0 {
		lastAssistantText = sb.String()
	}
}

func handleResult(line []byte) {
	var res ResultEvent
	if err := json.Unmarshal(line, &res); err != nil {
		return
	}
	if res.IsError {
		msg := res.Result
		if msg == "" {
			msg = res.Error
		}
		if msg != "" && msg != lastAssistantText {
			fmt.Printf("%s%s%s\n", Red, msg, Reset)
		}
	}
}

func printPrettyParams(jsonStr string) {
	var params map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
		return
	}

	for key, value := range params {
		fmt.Printf("  %s%s%s: ", Yellow, key, Reset)
		switch v := value.(type) {
		case string:
			fmt.Printf("%s%s%s\n", Green, v, Reset)
		case []any:
			fmt.Println()
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					printMapItem(m)
				} else {
					fmt.Printf("    %s-%s %s%v%s\n", Dim, Reset, Dim, item, Reset)
				}
			}
		default:
			out, _ := json.Marshal(v)
			fmt.Printf("%s%s%s\n", Magenta, string(out), Reset)
		}
	}
}

func printMapItem(m map[string]any) {
	// Special handling for todo items
	if content, hasContent := m["content"]; hasContent {
		status, _ := m["status"].(string)
		contentStr, _ := content.(string)

		var statusIcon, statusColor string
		switch status {
		case "completed":
			statusIcon = "✓"
			statusColor = Green
		case "in_progress":
			statusIcon = "→"
			statusColor = Cyan
		default:
			statusIcon = "○"
			statusColor = Dim
		}

		fmt.Printf("    %s%s%s %s%s%s\n", statusColor, statusIcon, Reset, statusColor, contentStr, Reset)
		return
	}

	// Generic map formatting: show key=value pairs
	fmt.Printf("    %s-%s ", Dim, Reset)
	first := true
	for k, v := range m {
		if !first {
			fmt.Printf("%s, %s", Dim, Reset)
		}
		first = false
		switch val := v.(type) {
		case string:
			fmt.Printf("%s%s%s=%s%s%s", Yellow, k, Reset, Green, val, Reset)
		default:
			fmt.Printf("%s%s%s=%s%v%s", Yellow, k, Reset, Magenta, val, Reset)
		}
	}
	fmt.Println()
}
