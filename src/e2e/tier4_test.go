package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTier4_Scenario1_CustomerFAQAutoReply(t *testing.T) {
	h := NewTestHarness(t)

	faqRules := []map[string]any{
		{"trigger_type": "contains", "trigger_value": "hours", "scope": "all", "response_type": "text", "response_content": "We are open Monday to Friday, 9am to 6pm."},
		{"trigger_type": "contains", "trigger_value": "refund", "scope": "all", "response_type": "text", "response_content": "Refund requests are processed within 3-5 business days."},
		{"trigger_type": "contains", "trigger_value": "location", "scope": "all", "response_type": "text", "response_content": "Our headquarters is located in Jakarta, Indonesia."},
	}

	for _, rule := range faqRules {
		var createResp APIResponse
		code, err := h.DoJSON(http.MethodPost, "/bot/rules", rule, &createResp)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, code)
	}

	customerQuestion := "Hi, could you please tell me your opening hours for tomorrow?"
	start := time.Now()
	rule, matched := h.MatchRule(customerQuestion, false)
	latency := time.Since(start).Milliseconds()

	require.True(t, matched)
	assert.Equal(t, "We are open Monday to Friday, 9am to 6pm.", rule.ResponseContent)

	h.Dispatcher.SendText("customer_01@s.whatsapp.net", rule.ResponseContent)
	assert.Len(t, h.Dispatcher.SentMessages, 1)

	err := h.LogEvent(BotEventLog{
		EventType:       "auto_reply",
		RuleID:          &rule.ID,
		SenderJID:       "customer_01@s.whatsapp.net",
		IncomingMessage: customerQuestion,
		ResponseMessage: rule.ResponseContent,
		LatencyMS:       latency,
		Status:          "success",
	})
	require.NoError(t, err)

	var logsResp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?event_type=auto_reply&status=success", nil, &logsResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := logsResp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), resMap["total"])
}

func TestTier4_Scenario2_GroupSpamShieldAntiLink(t *testing.T) {
	h := NewTestHarness(t)

	groupConfig := map[string]any{
		"group_jid":         "trading_community@g.us",
		"anti_link_enabled": true,
	}
	var setupResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", groupConfig, &setupResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	spamMessage := "Earn $500 daily guaranteed! Join here now: https://chat.whatsapp.com/CryptoRichFast"
	spammerJID := "spammer_account@s.whatsapp.net"

	start := time.Now()
	modRes, violated := h.ModerateMessage("trading_community@g.us", spammerJID, spamMessage)
	latency := time.Since(start).Milliseconds()

	require.True(t, violated)
	require.True(t, modRes.ShouldRevoke)

	h.Dispatcher.RevokeMessage("trading_community@g.us", spammerJID, "MSG-SPAM-999")
	assert.Len(t, h.Dispatcher.Revocations, 1)
	assert.Equal(t, "trading_community@g.us", h.Dispatcher.Revocations[0].GroupJID)
	assert.Equal(t, spammerJID, h.Dispatcher.Revocations[0].SenderJID)

	err = h.LogEvent(BotEventLog{
		EventType:       "group_moderation",
		SenderJID:       spammerJID,
		GroupJID:        "trading_community@g.us",
		IncomingMessage: spamMessage,
		ResponseMessage: "Message revoked due to anti-link policy",
		LatencyMS:       latency,
		Status:          "success",
	})
	require.NoError(t, err)

	var logCount int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE event_type = 'group_moderation' AND group_jid = 'trading_community@g.us'`).Scan(&logCount)
	require.NoError(t, err)
	assert.Equal(t, 1, logCount)
}

func TestTier4_Scenario3_GroupMemberOnboardingFarewell(t *testing.T) {
	h := NewTestHarness(t)

	onboardingConfig := map[string]any{
		"group_jid":         "vip_club@g.us",
		"welcome_enabled":   true,
		"welcome_template":  "Hello @{name}! Welcome to {group}. Please introduce yourself.",
		"farewell_enabled":  true,
		"farewell_template": "Farewell @{name}. Wishing you all the best from {group}.",
	}
	var setupResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", onboardingConfig, &setupResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	newMemberName := "David"
	groupName := "VIP Club"
	welcomeText := h.FormatWelcome("Hello @{name}! Welcome to {group}. Please introduce yourself.", newMemberName, groupName)
	assert.Equal(t, "Hello @David! Welcome to VIP Club. Please introduce yourself.", welcomeText)
	h.Dispatcher.SendText("vip_club@g.us", welcomeText)
	assert.Len(t, h.Dispatcher.SentMessages, 1)

	err = h.LogEvent(BotEventLog{
		EventType:       "group_moderation",
		SenderJID:       "david@s.whatsapp.net",
		GroupJID:        "vip_club@g.us",
		IncomingMessage: "EVENT_PARTICIPANT_JOIN",
		ResponseMessage: welcomeText,
		LatencyMS:       15,
		Status:          "success",
	})
	require.NoError(t, err)

	departingMemberName := "Emily"
	farewellText := h.FormatFarewell("Farewell @{name}. Wishing you all the best from {group}.", departingMemberName, groupName)
	assert.Equal(t, "Farewell @Emily. Wishing you all the best from VIP Club.", farewellText)
	h.Dispatcher.SendText("vip_club@g.us", farewellText)
	assert.Len(t, h.Dispatcher.SentMessages, 2)

	err = h.LogEvent(BotEventLog{
		EventType:       "group_moderation",
		SenderJID:       "emily@s.whatsapp.net",
		GroupJID:        "vip_club@g.us",
		IncomingMessage: "EVENT_PARTICIPANT_LEAVE",
		ResponseMessage: farewellText,
		LatencyMS:       12,
		Status:          "success",
	})
	require.NoError(t, err)

	var logCount int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE group_jid = 'vip_club@g.us'`).Scan(&logCount)
	require.NoError(t, err)
	assert.Equal(t, 2, logCount)
}

func TestTier4_Scenario4_AutonomousAIBotQueryTools(t *testing.T) {
	h := NewTestHarness(t)

	toolListReq := ToolRequest{
		Tool: "query_chats",
		Parameters: map[string]any{
			"action": "list_chats",
			"limit":  5,
		},
	}
	var toolResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", toolListReq, &toolResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	h.SetOpenAIResponse("list recent chats", "You have 2 active chats: Alice and Dev Team.", nil, http.StatusOK)

	chatReq := ChatRequest{
		Message: "!ai list recent chats",
	}
	var chatResp APIResponse
	code, err = h.DoJSON(http.MethodPost, "/bot/ai/chat", chatReq, &chatResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	resMap, ok := chatResp.Results.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, resMap["reply"], "You have 2 active chats")
}

func TestTier4_Scenario5_EndToEndRuleLifecycle(t *testing.T) {
	h := NewTestHarness(t)

	rulePayload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    "status",
		"scope":            "all",
		"response_type":    "text",
		"response_content": "All systems operational",
		"is_active":        true,
	}

	var createResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", rulePayload, &createResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, code)
	createResult := createResp.Results.(map[string]any)
	ruleID := int64(createResult["id"].(float64))

	rule, matched := h.MatchRule("status", false)
	require.True(t, matched)
	assert.Equal(t, "All systems operational", rule.ResponseContent)

	_ = h.LogEvent(BotEventLog{
		EventType:       "auto_reply",
		RuleID:          &ruleID,
		SenderJID:       "monitor@s.whatsapp.net",
		IncomingMessage: "status",
		ResponseMessage: rule.ResponseContent,
		LatencyMS:       10,
		Status:          "success",
	})

	var toggleResp APIResponse
	code, err = h.DoJSON(http.MethodPatch, fmt.Sprintf("/bot/rules/%d/toggle", ruleID), nil, &toggleResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	_, matched = h.MatchRule("status", false)
	assert.False(t, matched)

	var delResp APIResponse
	code, err = h.DoJSON(http.MethodDelete, fmt.Sprintf("/bot/rules/%d", ruleID), nil, &delResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var getResp APIResponse
	code, err = h.DoJSON(http.MethodGet, fmt.Sprintf("/bot/rules/%d", ruleID), nil, &getResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, code)

	var clearLogsResp APIResponse
	code, err = h.DoJSON(http.MethodDelete, "/bot/logs", nil, &clearLogsResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var logsCount int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs`).Scan(&logsCount)
	require.NoError(t, err)
	assert.Equal(t, 0, logsCount)
}

func TestTier4_Scenario6_AutonomousAIGroupAdminKick(t *testing.T) {
	h := NewTestHarness(t)

	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('governed_room@g.us', 1)`)
	require.NoError(t, err)

	maliciousMessage := "Click here for free bitcoin: https://chat.whatsapp.com/IllegalInvite"
	offenderJID := "malicious_bot@s.whatsapp.net"

	modRes, violated := h.ModerateMessage("governed_room@g.us", offenderJID, maliciousMessage)
	require.True(t, violated)
	require.True(t, modRes.ShouldRevoke)

	h.Dispatcher.RevokeMessage("governed_room@g.us", offenderJID, "OFFENSE-MSG-1")
	assert.Len(t, h.Dispatcher.Revocations, 1)

	toolReq := ToolRequest{
		Tool: "manage_group",
		Parameters: map[string]any{
			"action":       "remove",
			"group_jid":    "governed_room@g.us",
			"participants": []string{offenderJID},
		},
	}

	var toolResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", toolReq, &toolResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Len(t, h.Dispatcher.GroupActions, 1)
	assert.Equal(t, "remove", h.Dispatcher.GroupActions[0].Action)
	assert.Equal(t, "governed_room@g.us", h.Dispatcher.GroupActions[0].GroupJID)
	assert.Equal(t, []string{offenderJID}, h.Dispatcher.GroupActions[0].Participants)

	err = h.LogEvent(BotEventLog{
		EventType:       "ai_tool",
		SenderJID:       "autonomous_admin",
		GroupJID:        "governed_room@g.us",
		IncomingMessage: "manage_group:remove",
		ResponseMessage: fmt.Sprintf("Removed offender %s", offenderJID),
		LatencyMS:       42,
		Status:          "success",
	})
	require.NoError(t, err)

	var logCount int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE event_type = 'ai_tool' AND group_jid = 'governed_room@g.us'`).Scan(&logCount)
	require.NoError(t, err)
	assert.Equal(t, 1, logCount)
}
