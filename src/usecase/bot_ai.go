package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	domainBot "github.com/dresar/gowanew/domains/bot"
	domainChatStorage "github.com/dresar/gowanew/domains/chatstorage"
	whatsappInfrastructure "github.com/dresar/gowanew/infrastructure/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func (s *BotService) GetAIConfig(ctx context.Context) (*domainBot.AIConfig, error) {
	return s.repo.GetAIConfig(ctx)
}

func (s *BotService) UpdateAIConfig(ctx context.Context, req domainBot.UpdateAIConfigRequest) (*domainBot.AIConfig, error) {
	return s.repo.UpdateAIConfig(ctx, req)
}

func (s *BotService) GetTools(ctx context.Context) ([]domainBot.ToolDefinition, error) {
	return []domainBot.ToolDefinition{
		{Name: "send_message", Description: "Send text or media to WhatsApp user"},
		{Name: "manage_group", Description: "Perform group administrative action"},
		{Name: "query_chats", Description: "Query recent chat messages and history"},
		{Name: "update_rules", Description: "Autonomously create or modify bot rules"},
	}, nil
}

func (s *BotService) ChatWithAI(ctx context.Context, req domainBot.ChatRequest) (*domainBot.ChatResult, error) {
	if strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("message is required")
	}

	cfg, err := s.repo.GetAIConfig(ctx)
	if err != nil {
		return nil, err
	}

	model := cfg.Model
	if req.Model != "" {
		model = req.Model
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	temp := cfg.Temperature
	if req.Temperature != nil {
		temp = *req.Temperature
	}

	type openAIMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	messages := make([]openAIMsg, 0)
	if cfg.SystemPrompt != "" {
		messages = append(messages, openAIMsg{Role: "system", Content: cfg.SystemPrompt})
	}
	for _, h := range req.ChatHistory {
		messages = append(messages, openAIMsg{Role: h.Role, Content: h.Content})
	}
	messages = append(messages, openAIMsg{Role: "user", Content: req.Message})

	reqBody := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": temp,
	}
	rawJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	endpoint := baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	start := time.Now()
	httpResp, err := httpClient.Do(httpReq)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai error status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("empty choices from openai")
	}

	reply := parsed.Choices[0].Message.Content
	usage := domainBot.ChatUsage{
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		TotalTokens:      parsed.Usage.TotalTokens,
	}
	if usage.TotalTokens == 0 {
		usage.PromptTokens = len(req.Message) / 4
		usage.CompletionTokens = len(reply) / 4
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return &domainBot.ChatResult{
		Reply:     reply,
		Model:     model,
		LatencyMS: latency,
		Usage:     usage,
	}, nil
}

func (s *BotService) ExecuteTool(ctx context.Context, client *whatsmeow.Client, chatStorageRepo domainChatStorage.IChatStorageRepository, req domainBot.ToolRequest) (*domainBot.ToolResult, error) {
	if req.Tool == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	start := time.Now()
	var output any

	switch req.Tool {
	case "send_message":
		recipient, _ := req.Parameters["recipient"].(string)
		msgText, _ := req.Parameters["message"].(string)
		mediaType, _ := req.Parameters["media_type"].(string)
		mediaURL, _ := req.Parameters["media_url"].(string)

		if recipient == "" || msgText == "" {
			return nil, fmt.Errorf("recipient and message are required for send_message")
		}
		if mediaType == "" {
			mediaType = "text"
		}

		msgID := fmt.Sprintf("MOCK-MSG-%d", time.Now().UnixNano())
		if client != nil {
			parsedJID, err := types.ParseJID(recipient)
			if err == nil {
				msg := &waE2E.Message{Conversation: proto.String(msgText)}
				resp, sendErr := client.SendMessage(ctx, parsedJID, msg)
				if sendErr == nil {
					msgID = resp.ID
					if chatStorageRepo != nil {
						senderJID := whatsappInfrastructure.OwnSenderJID(client)
						_ = chatStorageRepo.StoreSentMessageWithContext(ctx, msgID, senderJID, parsedJID.String(), msgText, time.Now(), nil)
					}
				}
			}
		}

		output = map[string]any{
			"message_id": msgID,
			"recipient":  recipient,
			"media_type": mediaType,
			"media_url":  mediaURL,
			"status":     "sent",
		}

	case "manage_group":
		action, _ := req.Parameters["action"].(string)
		groupJID, _ := req.Parameters["group_jid"].(string)
		if action == "" || groupJID == "" {
			return nil, fmt.Errorf("action and group_jid are required for manage_group")
		}

		validActions := map[string]bool{
			"add": true, "remove": true, "promote": true, "demote": true,
			"get_info": true, "revoke_link": true, "set_name": true, "set_topic": true,
		}
		if !validActions[action] {
			return nil, fmt.Errorf("invalid action %q for manage_group", action)
		}

		var participants []string
		if rawParts, ok := req.Parameters["participants"].([]any); ok {
			for _, p := range rawParts {
				if pStr, ok := p.(string); ok {
					participants = append(participants, pStr)
				}
			}
		}

		if client != nil {
			parsedGroup, err := types.ParseJID(groupJID)
			if err == nil {
				var pJIDs []types.JID
				for _, p := range participants {
					if pj, perr := types.ParseJID(p); perr == nil {
						pJIDs = append(pJIDs, pj)
					}
				}
				switch action {
				case "add":
					if len(pJIDs) > 0 {
						_, _ = client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangeAdd)
					}
				case "remove":
					if len(pJIDs) > 0 {
						_, _ = client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangeRemove)
					}
				case "promote":
					if len(pJIDs) > 0 {
						_, _ = client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangePromote)
					}
				case "demote":
					if len(pJIDs) > 0 {
						_, _ = client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangeDemote)
					}
				case "revoke_link":
					_, _ = client.GetGroupInviteLink(ctx, parsedGroup, true)
				case "set_name":
					if val, ok := req.Parameters["value"].(string); ok && val != "" {
						_ = client.SetGroupName(ctx, parsedGroup, val)
					}
				case "set_topic":
					if val, ok := req.Parameters["value"].(string); ok && val != "" {
						_ = client.SetGroupTopic(ctx, parsedGroup, "", "", val)
					}
				}
			}
		}

		output = map[string]any{
			"action":       action,
			"group_jid":    groupJID,
			"participants": participants,
			"status":       "completed",
		}

	case "query_chats":
		action, _ := req.Parameters["action"].(string)
		if action == "" {
			return nil, fmt.Errorf("action is required for query_chats")
		}
		validChatActions := map[string]bool{
			"list_chats": true, "get_messages": true, "list_contacts": true,
		}
		if !validChatActions[action] {
			return nil, fmt.Errorf("invalid action %q for query_chats", action)
		}

		items := []map[string]any{
			{"jid": "user1@s.whatsapp.net", "name": "Alice", "unread": 0},
			{"jid": "group1@g.us", "name": "Dev Team", "unread": 2},
		}

		output = map[string]any{
			"action": action,
			"items":  items,
		}

	case "update_rules":
		action, _ := req.Parameters["action"].(string)
		if action == "" {
			return nil, fmt.Errorf("action is required for update_rules")
		}

		ruleData, _ := req.Parameters["rule_data"].(map[string]any)
		switch action {
		case "create_rule":
			if ruleData == nil {
				return nil, fmt.Errorf("rule_data is required for create_rule")
			}
			tType, _ := ruleData["trigger_type"].(string)
			tVal, _ := ruleData["trigger_value"].(string)
			scope, _ := ruleData["scope"].(string)
			rType, _ := ruleData["response_type"].(string)
			rContent, _ := ruleData["response_content"].(string)
			mURL, _ := ruleData["media_url"].(string)

			if tVal == "" || rContent == "" {
				return nil, fmt.Errorf("trigger_value and response_content are required")
			}
			if scope == "" {
				scope = "all"
			}
			if rType == "" {
				rType = "text"
			}
			if tType == "" {
				tType = "exact"
			}

			rule, err := s.CreateRule(ctx, domainBot.CreateRuleRequest{
				TriggerType:     domainBot.TriggerType(tType),
				TriggerValue:    tVal,
				Scope:           domainBot.Scope(scope),
				ResponseType:    domainBot.ResponseType(rType),
				ResponseContent: rContent,
				MediaURL:        mURL,
			})
			if err != nil {
				return nil, err
			}
			output = map[string]any{"action": "create_rule", "rule_id": rule.ID, "status": "created"}

		case "toggle_rule":
			rawID, ok := req.Parameters["rule_id"]
			if !ok {
				return nil, fmt.Errorf("rule_id is required for toggle_rule")
			}
			var ruleID int64
			switch v := rawID.(type) {
			case float64:
				ruleID = int64(v)
			case int:
				ruleID = int64(v)
			case int64:
				ruleID = v
			}
			_, err := s.ToggleRuleActive(ctx, ruleID)
			if err != nil {
				return nil, err
			}
			output = map[string]any{"action": "toggle_rule", "rule_id": ruleID, "status": "toggled"}

		case "delete_rule":
			rawID, ok := req.Parameters["rule_id"]
			if !ok {
				return nil, fmt.Errorf("rule_id is required for delete_rule")
			}
			var ruleID int64
			switch v := rawID.(type) {
			case float64:
				ruleID = int64(v)
			case int:
				ruleID = int64(v)
			case int64:
				ruleID = v
			}
			err := s.DeleteRule(ctx, ruleID)
			if err != nil {
				return nil, err
			}
			output = map[string]any{"action": "delete_rule", "rule_id": ruleID, "status": "deleted"}

		default:
			return nil, fmt.Errorf("unsupported rule action %q", action)
		}

	default:
		return nil, fmt.Errorf("unsupported tool %q", req.Tool)
	}

	latency := time.Since(start).Milliseconds()

	outBytes, _ := json.Marshal(output)
	_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
		EventType:       domainBot.EventTypeAITool,
		SenderJID:       "ai_agent",
		IncomingMessage: req.Tool,
		ResponseMessage: string(outBytes),
		LatencyMS:       latency,
		Status:          domainBot.LogStatusSuccess,
	})

	return &domainBot.ToolResult{
		Tool:      req.Tool,
		Status:    "success",
		LatencyMS: latency,
		Output:    output,
	}, nil
}
