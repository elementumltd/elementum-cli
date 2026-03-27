// Copyright 2026 Elementum Ltd. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/elementumltd/elementum-cli/auth"
	"github.com/elementumltd/elementum-cli/ui"
	"github.com/elementumltd/elementum-cli/internal/client"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat <agent-name-or-id>",
	Short: "Start a conversation with an Elementum agent",
	Long: `Start an interactive conversation with an Elementum agent.

The agent can be specified by name (case-insensitive) or UUID.

Examples:
  ei agents --app clm                                # List agents in the CLM app
  ei chat "IT Support Agent"                         # Chat with agent by name
  ei chat 794e1e48-73af-4760-8a1f-...               # Chat with agent by ID
  ei chat "IT Support Agent" -m "Hello"              # Send single message and exit
  ei chat "IT Support Agent" --list                  # List recent conversations
  ei chat "IT Support Agent" --continue <conv-id>    # Continue existing conversation

Multi-turn example:
  ei chat "IT Support Agent" -m "Hello"                              # Returns conversation ID
  ei chat "IT Support Agent" -c <conv-id> -m "Tell me more"         # Continue conversation`,
	Args: cobra.ExactArgs(1),
	RunE: runChat,
}

var (
	chatMessage        string
	chatConversationID string
	chatList           bool
)

func init() {
	chatCmd.Flags().StringVarP(&chatMessage, "message", "m", "", "Send a single message (non-interactive)")
	chatCmd.Flags().StringVarP(&chatConversationID, "continue", "c", "", "Continue an existing conversation")
	chatCmd.Flags().BoolVar(&chatList, "list", false, "List recent conversations for this agent")
}

// GetChatCmd returns the chat command for registration
func GetChatCmd() *cobra.Command {
	return chatCmd
}

func runChat(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get authenticated client
	apiClient, err := auth.GetClientFromCmd(cmd)
	if err != nil {
		return err
	}

	agentID, err := resolveAgentID(ctx, cmd, apiClient, args[0])
	if err != nil {
		return err
	}

	// Get agent details
	agentName, firstMessage, err := getAgentDetails(ctx, apiClient, agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent details: %w", err)
	}

	// List conversations mode
	if chatList {
		return listConversations(ctx, apiClient, agentID, agentName)
	}

	// Get or create conversation
	var convIDs ConversationIDs
	if chatConversationID != "" {
		// User provided an AgentConversation ID via --continue, look up the REST ID
		restID, err := lookupRestID(ctx, apiClient, agentID, chatConversationID)
		if err != nil {
			return fmt.Errorf("failed to look up conversation: %w", err)
		}
		convIDs = ConversationIDs{
			DisplayID: chatConversationID,
			RestID:    restID,
		}
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Continuing conversation %s...", chatConversationID)))
	} else {
		// Create new conversation
		ids, err := createConversation(ctx, apiClient, agentID)
		if err != nil {
			return fmt.Errorf("failed to create conversation: %w", err)
		}
		convIDs = ids
	}

	// Single message mode
	if chatMessage != "" {
		return sendSingleMessage(ctx, apiClient, convIDs, chatMessage)
	}

	// Interactive mode
	return runInteractiveChat(ctx, apiClient, agentID, agentName, convIDs, firstMessage)
}

func getAgentDetails(ctx context.Context, apiClient *client.Client, agentID string) (name, firstMessage string, err error) {
	resp, err := client.GetAgentById(ctx, apiClient.Genqlient(), agentID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get agent: %w", err)
	}

	if resp.Organization.Agent == nil {
		return "", "", fmt.Errorf("agent not found")
	}

	agent := *resp.Organization.Agent

	// Get firstMessage from the inline fragment (AgentElementum)
	var fm string
	if elemAgent, ok := agent.(*client.GetAgentByIdOrganizationAgentAgentElementum); ok {
		if elemAgent.FirstMessage != nil {
			fm = *elemAgent.FirstMessage
		}
	}

	return agent.GetName(), fm, nil
}

func looksLikeUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// ConversationIDs holds both the display ID (for --list compatibility) and REST ID (for API calls)
type ConversationIDs struct {
	DisplayID string // AgentConversation.Id - shown to user, used with --list/--continue
	RestID    string // Conversation.Id - used for REST API calls
}

func createConversation(ctx context.Context, apiClient *client.Client, agentID string) (ConversationIDs, error) {
	input := client.AgentConversationCreateInput{
		AgentId: agentID,
		Channel: "ELEMENTUM_UI",
	}

	resp, err := client.CreateAgentConversation(ctx, apiClient.Genqlient(), input)
	if err != nil {
		return ConversationIDs{}, err
	}

	return ConversationIDs{
		DisplayID: resp.AgentConversationCreateV2.Id,
		RestID:    resp.AgentConversationCreateV2.Conversation.Id,
	}, nil
}

// lookupRestID converts an AgentConversation ID (from --continue) to the REST API conversation ID
func lookupRestID(ctx context.Context, apiClient *client.Client, agentID, displayID string) (string, error) {
	resp, err := client.GetConversationRestId(ctx, apiClient.Genqlient(), agentID, displayID)
	if err != nil {
		return "", fmt.Errorf("failed to look up conversation: %w", err)
	}

	if resp.Organization.Agent == nil {
		return "", fmt.Errorf("agent not found")
	}

	// Agent is a pointer to interface, dereference and call method
	agent := *resp.Organization.Agent
	conv := agent.GetConversationV2()
	if conv == nil {
		return "", fmt.Errorf("conversation not found")
	}

	return conv.Conversation.Id, nil
}

func listConversations(ctx context.Context, apiClient *client.Client, agentID, agentName string) error {
	resp, err := client.ListAgentConversations(ctx, apiClient.Genqlient(), agentID)
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	if resp.Organization.Agent == nil {
		return fmt.Errorf("agent not found")
	}

	agent := *resp.Organization.Agent
	conversations := agent.GetConversationsV2().Edges

	if len(conversations) == 0 {
		fmt.Println()
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("No conversations found for agent %q.", agentName)))
		fmt.Println()
		return nil
	}

	table := ui.NewTable([]string{"TITLE", "UPDATED", "ID"})

	for _, edge := range conversations {
		conv := edge.Node
		updated := conv.UpdatedAt
		title := "(untitled)"
		if conv.Title != nil && *conv.Title != "" {
			title = *conv.Title
		}
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		table.AddRow(title, updated, conv.Id)
	}

	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Conversations with %s", agentName)))
	fmt.Println()
	fmt.Println(table.Render())
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("Use 'ei chat <agent-name-or-id> --continue <id>' to continue a conversation"))
	fmt.Println()

	return nil
}

// SSE Message types
type SSEEvent struct {
	Event string
	Data  string
}

type AgentMessageRequest struct {
	ConversationID string               `json:"conversationId"`
	Message        string               `json:"message"`
	Context        *AgentMessageContext `json:"context,omitempty"`
}

type AgentMessageContext struct {
	ChannelInstructions string `json:"channelInstructions,omitempty"`
}

type SSEMessageData struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Tool    *struct {
		Name        string `json:"name,omitempty"`
		DisplayName string `json:"displayName,omitempty"`
	} `json:"tool,omitempty"`
	Error string `json:"error,omitempty"`
}

func sendSingleMessage(ctx context.Context, apiClient *client.Client, convIDs ConversationIDs, message string) error {
	fmt.Println()
	fmt.Print(ui.LabelStyle.Render("You: "))
	fmt.Println(message)
	fmt.Println()
	fmt.Print(ui.InfoStyle.Render("Agent: "))

	start := time.Now()
	// Use RestID for the actual API call
	err := streamAgentMessage(ctx, apiClient, convIDs.RestID, message, func(event SSEEvent) error {
		return handleSSEEvent(event, true)
	})
	elapsed := time.Since(start)

	fmt.Println()
	fmt.Println()

	// Print DisplayID (AgentConversation.Id) - this matches what --list shows
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Conversation ID: %s", convIDs.DisplayID)))
	fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("Response time: %s", elapsed.Round(time.Millisecond))))
	fmt.Println(ui.MutedStyle.Render("Continue with: ei chat <agent-id> -c " + convIDs.DisplayID + " -m \"your message\""))
	fmt.Println(ui.MutedStyle.Render("Analyze with:  ei conversation <agent-id> " + convIDs.DisplayID))
	fmt.Println()

	return err
}

func runInteractiveChat(ctx context.Context, apiClient *client.Client, agentID, agentName string, convIDs ConversationIDs, firstMessage string) error {
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("Chat with %s", agentName)))
	fmt.Println(ui.MutedStyle.Render("Type your message and press Enter. Type /quit to exit, /new to start a new conversation."))
	fmt.Println()

	// Show first message if available
	if firstMessage != "" {
		fmt.Print(ui.InfoStyle.Render("Agent: "))
		fmt.Println(firstMessage)
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(ui.LabelStyle.Render("You: "))

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle commands
		switch strings.ToLower(input) {
		case "/quit", "/exit", "/q":
			fmt.Println(ui.MutedStyle.Render("Goodbye!"))
			return nil
		case "/new":
			newConvIDs, err := createConversation(ctx, apiClient, agentID)
			if err != nil {
				fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Failed to create new conversation: %v", err)))
				continue
			}
			convIDs = newConvIDs
			fmt.Println(ui.MutedStyle.Render("Started new conversation."))
			fmt.Println()
			continue
		case "/help":
			fmt.Println(ui.MutedStyle.Render(`
Commands:
  /quit, /exit, /q  - Exit the chat
  /new              - Start a new conversation
  /paste            - Enter multiline input mode (end with --- on its own line)
  /file <path>      - Send contents of a file
  /help             - Show this help
`))
			continue
		case "/paste":
			fmt.Println(ui.MutedStyle.Render("Enter multiline text. Type --- on its own line to send:"))
			var lines []string
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("failed to read input: %w", err)
				}
				trimmed := strings.TrimSpace(line)
				if trimmed == "---" {
					break
				}
				lines = append(lines, strings.TrimRight(line, "\n"))
			}
			input = strings.Join(lines, "\n")
			if strings.TrimSpace(input) == "" {
				fmt.Println(ui.MutedStyle.Render("(empty input, cancelled)"))
				continue
			}
			fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("(sending %d lines, %d chars)", len(lines), len(input))))
		}

		// Handle /file command
		if strings.HasPrefix(strings.ToLower(input), "/file ") {
			filePath := strings.TrimSpace(input[6:])
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Failed to read file: %v", err)))
				continue
			}
			input = string(content)
			fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("(sending file: %d chars)", len(input))))
		}

		// Send message using RestID for the API call
		fmt.Println()
		fmt.Print(ui.InfoStyle.Render("Agent: "))

		start := time.Now()
		err = streamAgentMessage(ctx, apiClient, convIDs.RestID, input, func(event SSEEvent) error {
			return handleSSEEvent(event, true)
		})
		elapsed := time.Since(start)

		if err != nil {
			fmt.Println()
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", err)))
		}

		fmt.Println()
		fmt.Println(ui.MutedStyle.Render(fmt.Sprintf("(%s)", elapsed.Round(time.Millisecond))))
		fmt.Println()
	}
}

func streamAgentMessage(ctx context.Context, apiClient *client.Client, conversationID, message string, handler func(SSEEvent) error) error {
	// Get access token
	token, err := apiClient.GetAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Build request
	reqBody := AgentMessageRequest{
		ConversationID: conversationID,
		Message:        message,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	baseURL := apiClient.GetRESTBaseURL()
	url := baseURL + "/api/v1/agent-conversations"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)

	// Set Elementum headers
	org := apiClient.GetOrganization()
	instance := apiClient.GetInstance()
	host := getHostForInstance(instance)
	req.Header.Set("x-elementum-organization", fmt.Sprintf("%s.%s", org, host))
	req.Header.Set("x-elementum-platform", "IOS")
	req.Header.Set("x-elementum-toe", uuid.New().String())
	req.Header.Set("x-elementum-base-url", host)

	// Create HTTP client with longer timeout for streaming
	httpClient := &http.Client{Timeout: 5 * time.Minute}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s\nURL: %s\nRequest: %s", resp.StatusCode, string(body), url, string(jsonBody))
	}

	// Read SSE stream
	reader := bufio.NewReader(resp.Body)
	var currentEvent SSEEvent

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = strings.TrimSpace(line)

		if line == "" {
			// Empty line means end of event
			if currentEvent.Data != "" {
				if err := handler(currentEvent); err != nil {
					return err
				}
			}
			currentEvent = SSEEvent{}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			currentEvent.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			currentEvent.Data = data
		}
	}
}

func handleSSEEvent(event SSEEvent, showTools bool) error {
	if event.Data == "" {
		return nil
	}

	// Debug: print raw event (uncomment for debugging)
	// fmt.Fprintf(os.Stderr, "[DEBUG] Event: %q Data: %s\n", event.Event, event.Data)

	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
		// Not JSON, just print it
		fmt.Print(event.Data)
		return nil
	}

	// Get message type from data
	msgType, _ := data["type"].(string)

	// Handle different message types based on data.type field
	switch msgType {
	case "UPDATE":
		// Text update - print the text chunk
		if text, ok := data["text"].(string); ok {
			fmt.Print(text)
		}
	case "START":
		// Conversation started, no output needed
	case "COMPLETE":
		// Stream complete
	case "TOOL_CALL_START":
		if showTools {
			toolName := ""
			if name, ok := data["toolDisplayName"].(string); ok && name != "" {
				toolName = name
			} else if name, ok := data["toolName"].(string); ok {
				toolName = name
			}
			if toolName != "" {
				fmt.Println()
				fmt.Print(ui.MutedStyle.Render(fmt.Sprintf("  [Using %s...]", toolName)))
			}
		}
	case "TOOL_CALL_COMPLETE":
		// Tool completed, no output needed
		if showTools {
			fmt.Println()
		}
	case "TOOL_CALL_ERROR":
		if errMsg, ok := data["error"].(string); ok && errMsg != "" {
			fmt.Println()
			fmt.Print(ui.WarningStyle.Render(fmt.Sprintf("  [Tool error: %s]", errMsg)))
			fmt.Println()
		}
	case "ERROR":
		if errMsg, ok := data["error"].(string); ok && errMsg != "" {
			return fmt.Errorf("agent error: %s", errMsg)
		}
	default:
		// For backward compatibility, also check event.Event for older format
		switch event.Event {
		case "text":
			if content, ok := data["content"].(string); ok {
				fmt.Print(content)
			}
		case "tool_start":
			if showTools {
				toolName := ""
				if tool, ok := data["tool"].(map[string]interface{}); ok {
					if name, ok := tool["displayName"].(string); ok && name != "" {
						toolName = name
					} else if name, ok := tool["name"].(string); ok {
						toolName = name
					}
				}
				if toolName != "" {
					fmt.Println()
					fmt.Print(ui.MutedStyle.Render(fmt.Sprintf("  [Using %s...]", toolName)))
					fmt.Println()
				}
			}
		}
	}

	return nil
}

func getHostForInstance(instance client.Instance) string {
	switch instance {
	case client.US, "production", "":
		return "elementum.io"
	default:
		return fmt.Sprintf("%s.elementum.io", instance)
	}
}
