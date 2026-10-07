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

	"github.com/dresar/gowanew/config"
	domainBot "github.com/dresar/gowanew/domains/bot"
	domainChatStorage "github.com/dresar/gowanew/domains/chatstorage"
	whatsappInfrastructure "github.com/dresar/gowanew/infrastructure/whatsapp"
	pkgUtils "github.com/dresar/gowanew/pkg/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func (s *BotService) GetAIConfig(ctx context.Context) (*domainBot.AIConfig, error) {
	cfg, err := s.repo.GetAIConfig(ctx)
	if err != nil && err != domainBot.ErrAIConfigNotFound {
		return nil, err
	}
	if cfg == nil {
		cfg = &domainBot.AIConfig{
			ID:               1,
			Provider:         "openai",
			BaseURL:          "https://api.openai.com/v1",
			APIKey:           "",
			Model:            "gpt-4o-mini",
			SystemPrompt:     "",
			Temperature:      0.7,
			TriggerPrefix:    "!ai",
			AutoReplyEnabled: false,
		}
	}
	if config.BotAIProvider != "" {
		cfg.Provider = config.BotAIProvider
	}
	if config.BotAIBaseURL != "" {
		cfg.BaseURL = config.BotAIBaseURL
	}
	if config.BotAIAPIKey != "" {
		cfg.APIKey = config.BotAIAPIKey
	}
	if config.BotAIModel != "" {
		cfg.Model = config.BotAIModel
	}
	if config.BotAISystemPrompt != "" {
		cfg.SystemPrompt = config.BotAISystemPrompt
	}
	if config.BotAITemperature > 0 {
		cfg.Temperature = config.BotAITemperature
	}
	if config.BotAITriggerPrefix != "" {
		cfg.TriggerPrefix = config.BotAITriggerPrefix
	}
	if config.BotAIAutoReplyEnabled {
		cfg.AutoReplyEnabled = true
	}
	return cfg, nil
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

const defaultAntiSlopPrompt = `Kamu adalah asisten pribadi yang membalas chat WhatsApp dengan gaya super santai, non-formal, luwes, dan seperti teman akrab orang Indonesia sehari-hari.
Aturan penting:
- Gunakan bahasa percakapan sehari-hari (contoh: "halo", "oke sip", "siap", "bentar ya", "ada apa nih?", "kenapa bro?", "santai aja", "gas").
- DILARANG KERAS berbicara kaku seperti bot, customer service, atau AI formal (JANGAN gunakan: "Tentu saja!", "Sebagai asisten AI", "Ada yang bisa saya bantu?", "Semoga membantu", "Halo, bagaimana kabarmu?").
- Jika lawan bicara bertanya apakah kamu kenal dia (misal: "kamu kenal aku?"), jawab secara natural dan santai bahwa kamu kenal dia dari nama/username profil WhatsApp miliknya.
- Jawab to the point, singkat, natural, tanpa pembuka atau penutup bertele-tele.`

func normalizePhoneNumber(val string) string {
	clean := strings.TrimSuffix(val, "@s.whatsapp.net")
	clean = strings.TrimSuffix(clean, "@g.us")
	clean = strings.TrimSuffix(clean, "@lid")
	var b strings.Builder
	for _, r := range clean {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if strings.HasPrefix(digits, "08") {
		digits = "62" + digits[1:]
	}
	return digits
}

func sanitizeContainerTag(jid string) string {
	clean := strings.TrimSuffix(jid, "@s.whatsapp.net")
	clean = strings.TrimSuffix(clean, "@g.us")
	clean = strings.TrimSuffix(clean, "@lid")
	var b strings.Builder
	for _, r := range clean {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == ':' {
			b.WriteRune(r)
		}
	}
	tag := b.String()
	if tag == "" {
		tag = "general"
	}
	return "user_" + tag
}

func querySupermemory(ctx context.Context, apiKey, tag, query string) string {
	if apiKey == "" || tag == "" || strings.TrimSpace(query) == "" {
		return ""
	}
	reqBody := map[string]any{
		"q":            query,
		"containerTag": tag,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return ""
	}

	searchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(searchCtx, http.MethodPost, "https://api.supermemory.ai/v4/search", bytes.NewReader(raw))
	if err != nil {
		return ""
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var res struct {
		Results []struct {
			Content string `json:"content"`
			Memory  string `json:"memory"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return ""
	}

	var sb strings.Builder
	for _, item := range res.Results {
		text := item.Content
		if text == "" {
			text = item.Memory
		}
		if text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n- ")
			} else {
				sb.WriteString("- ")
			}
			sb.WriteString(strings.TrimSpace(text))
		}
	}
	return sb.String()
}

func ingestSupermemoryAsync(apiKey, tag, userMsg, assistantReply string) {
	if apiKey == "" || tag == "" || strings.TrimSpace(userMsg) == "" {
		return
	}
	go func() {
		convText := fmt.Sprintf("User: %s\nAsisten: %s", strings.TrimSpace(userMsg), strings.TrimSpace(assistantReply))
		reqBody := map[string]any{
			"content":      convText,
			"containerTag": tag,
		}
		raw, err := json.Marshal(reqBody)
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.supermemory.ai/v3/documents", bytes.NewReader(raw))
		if err != nil {
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(httpReq)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}

func (s *BotService) ChatWithAI(ctx context.Context, req domainBot.ChatRequest) (*domainBot.ChatResult, error) {
	if strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("message is required")
	}

	cfg, err := s.GetAIConfig(ctx)
	if err != nil {
		return nil, err
	}

	model := cfg.Model
	if req.Model != "" {
		model = req.Model
	}
	if model == "" {
		model = "step-5-preview"
	}

	temp := cfg.Temperature
	if req.Temperature != nil {
		temp = *req.Temperature
	}

	sysPrompt := strings.TrimSpace(cfg.SystemPrompt)
	if sysPrompt == "" {
		sysPrompt = defaultAntiSlopPrompt
	}

	senderPhone := normalizePhoneNumber(req.SenderJID)
	var matchedPersona *domainBot.AIPersona
	if senderPhone != "" {
		p, pErr := s.repo.GetAIPersonaByPhone(ctx, senderPhone)
		if pErr == nil && p != nil && p.IsActive {
			matchedPersona = p
		}
	}

	tag := ""
	if req.SenderJID != "" {
		tag = sanitizeContainerTag(req.SenderJID)
	}

	if matchedPersona != nil {
		if strings.TrimSpace(matchedPersona.CustomPrompt) != "" {
			sysPrompt = strings.TrimSpace(matchedPersona.CustomPrompt)
		}
		if matchedPersona.ContactName != "" {
			rel := matchedPersona.Relationship
			if rel == "" {
				rel = "teman"
			}
			sysPrompt += fmt.Sprintf("\n\nLawan bicaramu bernama: %s (Relasi: %s). Perlakukan dia sesuai dengan relasi ini.", matchedPersona.ContactName, rel)
		}
		sysPrompt += "\n\nInstruksi Khusus Gaya Komunikasi:\n- Balaslah dengan bahasa super natural, santai, ekspresif, tidak kaku, manusiawi seutuhnya, dan selaras dengan riwayat percakapan sebelumnya.\n- Dilarang keras terdengar seperti AI, bot, robot, atau asisten virtual formal."
	} else {
		if req.SenderName != "" {
			sysPrompt += fmt.Sprintf("\n\nLawan bicaramu saat ini memiliki nama/username WhatsApp: %s. Kamu tahu namanya dari data profil WhatsApp dia.", req.SenderName)
		}

		if strings.Contains(req.SenderJID, "6285216149732") || strings.Contains(req.SenderName, "Indah") {
			sysPrompt += "\n\nPerhatian Khusus: Kontak ini adalah Indah (+62 852-1614-9732), pacar tercinta dari Eka Syarif Maulana. Balaslah dengan gaya obrolan sehari-hari pasangan: sangat hangat, manis, perhatian (tanya makan, kabar, jangan begadang), akrab, santai, dan penuh kasih sayang. Dilarang keras bersikap kaku atau formal seperti robot/CS."
		}
	}

	useMemory := true
	if matchedPersona != nil && !matchedPersona.UseMemory {
		useMemory = false
	}

	if useMemory && config.SupermemoryEnabled && config.SupermemoryAPIKey != "" && tag != "" {
		memContext := querySupermemory(ctx, config.SupermemoryAPIKey, tag, req.Message)
		if memContext != "" {
			sysPrompt += "\n\n[MEMORI KONTEKS PERCAKAPAN SEBELUMNYA DENGAN KONTAK INI]:\n" + memContext
		}
	}

	type openAIMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	messages := make([]openAIMsg, 0)
	messages = append(messages, openAIMsg{Role: "system", Content: sysPrompt})
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
		baseURL = "https://router.bynara.id/v1"
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

	if useMemory && config.SupermemoryEnabled && config.SupermemoryAPIKey != "" && tag != "" && reply != "" {
		ingestSupermemoryAsync(config.SupermemoryAPIKey, tag, req.Message, reply)
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
			parsedJID, err := pkgUtils.ParseJID(recipient)
			if err != nil {
				return nil, fmt.Errorf("invalid recipient JID %q: %w", recipient, err)
			}
			msg := &waE2E.Message{Conversation: proto.String(msgText)}
			resp, sendErr := client.SendMessage(ctx, parsedJID, msg)
			if sendErr != nil {
				return nil, fmt.Errorf("failed to send message: %w", sendErr)
			}
			msgID = resp.ID
			if chatStorageRepo != nil {
				senderJID := whatsappInfrastructure.OwnSenderJID(client)
				_ = chatStorageRepo.StoreSentMessageWithContext(ctx, msgID, senderJID, parsedJID.String(), msgText, time.Now(), nil)
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

		groupDetails := map[string]any{}
		if client != nil {
			parsedGroup, err := pkgUtils.ParseJID(groupJID)
			if err != nil {
				return nil, fmt.Errorf("invalid group JID %q: %w", groupJID, err)
			}

			var pJIDs []types.JID
			for _, p := range participants {
				if pj, perr := pkgUtils.ParseJID(p); perr == nil {
					pJIDs = append(pJIDs, pj)
				}
			}

			switch action {
			case "get_info":
				info, err := client.GetGroupInfo(ctx, parsedGroup)
				if err != nil {
					return nil, fmt.Errorf("failed to get group info: %w", err)
				}
				partList := make([]map[string]any, 0, len(info.Participants))
				for _, p := range info.Participants {
					partList = append(partList, map[string]any{
						"jid":            p.JID.String(),
						"is_admin":       p.IsAdmin,
						"is_super_admin": p.IsSuperAdmin,
					})
				}
				groupDetails["name"] = info.GroupName.Name
				groupDetails["topic"] = info.Topic
				groupDetails["participants_detail"] = partList
			case "add":
				if len(pJIDs) > 0 {
					if _, err := client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangeAdd); err != nil {
						return nil, fmt.Errorf("failed to add participants: %w", err)
					}
				}
			case "remove":
				if len(pJIDs) > 0 {
					if _, err := client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangeRemove); err != nil {
						return nil, fmt.Errorf("failed to remove participants: %w", err)
					}
				}
			case "promote":
				if len(pJIDs) > 0 {
					if _, err := client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangePromote); err != nil {
						return nil, fmt.Errorf("failed to promote participants: %w", err)
					}
				}
			case "demote":
				if len(pJIDs) > 0 {
					if _, err := client.UpdateGroupParticipants(ctx, parsedGroup, pJIDs, whatsmeow.ParticipantChangeDemote); err != nil {
						return nil, fmt.Errorf("failed to demote participants: %w", err)
					}
				}
			case "revoke_link":
				link, err := client.GetGroupInviteLink(ctx, parsedGroup, true)
				if err != nil {
					return nil, fmt.Errorf("failed to revoke invite link: %w", err)
				}
				groupDetails["new_invite_link"] = link
			case "set_name":
				if val, ok := req.Parameters["value"].(string); ok && val != "" {
					if err := client.SetGroupName(ctx, parsedGroup, val); err != nil {
						return nil, fmt.Errorf("failed to set group name: %w", err)
					}
				}
			case "set_topic":
				if val, ok := req.Parameters["value"].(string); ok && val != "" {
					if err := client.SetGroupTopic(ctx, parsedGroup, "", "", val); err != nil {
						return nil, fmt.Errorf("failed to set group topic: %w", err)
					}
				}
			}
		}

		outMap := map[string]any{
			"action":       action,
			"group_jid":    groupJID,
			"participants": participants,
			"status":       "completed",
		}
		for k, v := range groupDetails {
			outMap[k] = v
		}
		output = outMap

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

		items := make([]map[string]any, 0)
		switch action {
		case "list_chats":
			if chatStorageRepo != nil {
				var filter domainChatStorage.ChatFilter
				if limit, ok := req.Parameters["limit"].(float64); ok && limit > 0 {
					filter.Limit = int(limit)
				} else if limitInt, ok := req.Parameters["limit"].(int); ok && limitInt > 0 {
					filter.Limit = limitInt
				}
				if q, ok := req.Parameters["query"].(string); ok && q != "" {
					filter.SearchName = q
				}
				chats, err := chatStorageRepo.GetChats(&filter)
				if err == nil {
					for _, c := range chats {
						items = append(items, map[string]any{
							"jid":               c.JID,
							"name":              c.Name,
							"last_message_time": c.LastMessageTime.Format(time.RFC3339),
							"archived":          c.Archived,
						})
					}
				}
			}
		case "get_messages":
			chatJID, _ := req.Parameters["chat_jid"].(string)
			if chatJID == "" {
				chatJID, _ = req.Parameters["jid"].(string)
			}
			if chatStorageRepo != nil && chatJID != "" {
				var filter domainChatStorage.MessageFilter
				filter.ChatJID = chatJID
				filter.Limit = 50
				if limit, ok := req.Parameters["limit"].(float64); ok && limit > 0 {
					filter.Limit = int(limit)
				} else if limitInt, ok := req.Parameters["limit"].(int); ok && limitInt > 0 {
					filter.Limit = limitInt
				}
				msgs, err := chatStorageRepo.GetMessages(&filter)
				if err == nil {
					for _, m := range msgs {
						items = append(items, map[string]any{
							"id":         m.ID,
							"sender":     m.Sender,
							"content":    m.Content,
							"timestamp":  m.Timestamp.Format(time.RFC3339),
							"is_from_me": m.IsFromMe,
							"media_type": m.MediaType,
						})
					}
				}
			}
		case "list_contacts":
			if client != nil && client.Store != nil && client.Store.Contacts != nil {
				contacts, err := client.Store.Contacts.GetAllContacts(ctx)
				if err == nil {
					for jid, contact := range contacts {
						name := contact.FullName
						if name == "" {
							name = contact.PushName
						}
						if name == "" {
							name = contact.BusinessName
						}
						items = append(items, map[string]any{
							"jid":  jid.String(),
							"name": name,
						})
					}
				}
			}
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

		case "update_rule":
			rawID, ok := req.Parameters["rule_id"]
			if !ok {
				return nil, fmt.Errorf("rule_id is required for update_rule")
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
			if ruleData == nil {
				return nil, fmt.Errorf("rule_data is required for update_rule")
			}
			var updateReq domainBot.UpdateRuleRequest
			if tType, ok := ruleData["trigger_type"].(string); ok && tType != "" {
				tt := domainBot.TriggerType(tType)
				updateReq.TriggerType = &tt
			}
			if tVal, ok := ruleData["trigger_value"].(string); ok {
				updateReq.TriggerValue = &tVal
			}
			if scope, ok := ruleData["scope"].(string); ok && scope != "" {
				sc := domainBot.Scope(scope)
				updateReq.Scope = &sc
			}
			if rType, ok := ruleData["response_type"].(string); ok && rType != "" {
				rt := domainBot.ResponseType(rType)
				updateReq.ResponseType = &rt
			}
			if rContent, ok := ruleData["response_content"].(string); ok {
				updateReq.ResponseContent = &rContent
			}
			if mURL, ok := ruleData["media_url"].(string); ok {
				updateReq.MediaURL = &mURL
			}
			if isActive, ok := ruleData["is_active"].(bool); ok {
				updateReq.IsActive = &isActive
			}
			rule, err := s.UpdateRule(ctx, ruleID, updateReq)
			if err != nil {
				return nil, err
			}
			output = map[string]any{"action": "update_rule", "rule": rule, "status": "updated"}

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

func (s *BotService) ListAIPersonas(ctx context.Context) ([]*domainBot.AIPersona, error) {
	return s.repo.ListAIPersonas(ctx)
}

func (s *BotService) GetAIPersonaByID(ctx context.Context, id int64) (*domainBot.AIPersona, error) {
	return s.repo.GetAIPersonaByID(ctx, id)
}

func (s *BotService) GetAIPersonaByPhone(ctx context.Context, phone string) (*domainBot.AIPersona, error) {
	cleanPhone := normalizePhoneNumber(phone)
	if cleanPhone == "" {
		cleanPhone = phone
	}
	return s.repo.GetAIPersonaByPhone(ctx, cleanPhone)
}

func (s *BotService) CreateAIPersona(ctx context.Context, req domainBot.CreateAIPersonaRequest) (*domainBot.AIPersona, error) {
	cleanPhone := normalizePhoneNumber(req.PhoneNumber)
	if cleanPhone == "" {
		return nil, domainBot.ErrMissingPhoneNumber
	}
	if strings.TrimSpace(req.CustomPrompt) == "" {
		return nil, domainBot.ErrMissingCustomPrompt
	}

	autoReply := true
	if req.AutoReplyEnabled != nil {
		autoReply = *req.AutoReplyEnabled
	}
	useMemory := true
	if req.UseMemory != nil {
		useMemory = *req.UseMemory
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	persona := &domainBot.AIPersona{
		PhoneNumber:      cleanPhone,
		ContactName:      strings.TrimSpace(req.ContactName),
		Relationship:     strings.TrimSpace(req.Relationship),
		CustomPrompt:     strings.TrimSpace(req.CustomPrompt),
		AutoReplyEnabled: autoReply,
		UseMemory:        useMemory,
		IsActive:         isActive,
	}
	return s.repo.CreateAIPersona(ctx, persona)
}

func (s *BotService) UpdateAIPersona(ctx context.Context, id int64, req domainBot.UpdateAIPersonaRequest) (*domainBot.AIPersona, error) {
	if req.PhoneNumber != nil {
		cleanPhone := normalizePhoneNumber(*req.PhoneNumber)
		if cleanPhone == "" {
			return nil, domainBot.ErrMissingPhoneNumber
		}
		req.PhoneNumber = &cleanPhone
	}
	if req.CustomPrompt != nil {
		trimmed := strings.TrimSpace(*req.CustomPrompt)
		if trimmed == "" {
			return nil, domainBot.ErrMissingCustomPrompt
		}
		req.CustomPrompt = &trimmed
	}
	return s.repo.UpdateAIPersona(ctx, id, req)
}

func (s *BotService) DeleteAIPersona(ctx context.Context, id int64) error {
	return s.repo.DeleteAIPersona(ctx, id)
}
