package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTier3_Pairwise_AutoResponder_EventLogger(t *testing.T) {
	h := NewTestHarness(t)
	res, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'order_status', 'all', 'text', 'Your order is shipped', 1)`)
	require.NoError(t, err)
	ruleID, _ := res.LastInsertId()

	start := time.Now()
	rule, matched := h.MatchRule("order_status", false)
	latency := time.Since(start).Milliseconds()
	require.True(t, matched)

	err = h.LogEvent(BotEventLog{
		EventType:       "auto_reply",
		RuleID:          &ruleID,
		SenderJID:       "buyer@s.whatsapp.net",
		IncomingMessage: "order_status",
		ResponseMessage: rule.ResponseContent,
		LatencyMS:       latency,
		Status:          "success",
	})
	require.NoError(t, err)

	var logCount int
	var loggedResp string
	err = h.DB.QueryRow(`SELECT COUNT(*), response_message FROM bot_event_logs WHERE rule_id = ?`, ruleID).Scan(&logCount, &loggedResp)
	require.NoError(t, err)
	assert.Equal(t, 1, logCount)
	assert.Equal(t, "Your order is shipped", loggedResp)
}

func TestTier3_Pairwise_GroupModeration_EventLogger(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('pair_group@g.us', 1)`)
	require.NoError(t, err)

	res, violated := h.ModerateMessage("pair_group@g.us", "bad_actor@s.whatsapp.net", "visit https://chat.whatsapp.com/SpamInvite")
	require.True(t, violated)

	if res.ShouldRevoke {
		h.Dispatcher.RevokeMessage("pair_group@g.us", "bad_actor@s.whatsapp.net", "REVOKE-PAIR-1")
		err = h.LogEvent(BotEventLog{
			EventType:       "group_moderation",
			SenderJID:       "bad_actor@s.whatsapp.net",
			GroupJID:        "pair_group@g.us",
			IncomingMessage: "visit https://chat.whatsapp.com/SpamInvite",
			ResponseMessage: "Message revoked due to anti-link policy",
			LatencyMS:       18,
			Status:          "success",
		})
		require.NoError(t, err)
	}

	assert.Len(t, h.Dispatcher.Revocations, 1)
	var logCount int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE event_type = 'group_moderation' AND group_jid = 'pair_group@g.us'`).Scan(&logCount)
	require.NoError(t, err)
	assert.Equal(t, 1, logCount)
}

func TestTier3_Pairwise_AIAssistant_AutonomousTools(t *testing.T) {
	h := NewTestHarness(t)
	toolCallPayload := []any{
		map[string]any{
			"id":   "call_123",
			"type": "function",
			"function": map[string]any{
				"name": "send_message",
				"arguments": `{"recipient": "client@s.whatsapp.net", "message": "Automated delivery update"}`,
			},
		},
	}
	h.SetOpenAIResponse("dispatch message", "I will send the delivery update now.", toolCallPayload, http.StatusOK)

	var chatResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", ChatRequest{Message: "dispatch message"}, &chatResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	toolReq := ToolRequest{
		Tool: "send_message",
		Parameters: map[string]any{
			"recipient": "client@s.whatsapp.net",
			"message":   "Automated delivery update",
		},
	}
	var toolResp APIResponse
	code, err = h.DoJSON(http.MethodPost, "/bot/ai/tools", toolReq, &toolResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Len(t, h.Dispatcher.SentMessages, 1)
	assert.Equal(t, "client@s.whatsapp.net", h.Dispatcher.SentMessages[0].Recipient)
	assert.Equal(t, "Automated delivery update", h.Dispatcher.SentMessages[0].Message)
}

func TestTier3_Pairwise_AutonomousTools_AutoResponderRules(t *testing.T) {
	h := NewTestHarness(t)
	toolReq := ToolRequest{
		Tool: "update_rules",
		Parameters: map[string]any{
			"action": "create_rule",
			"rule_data": map[string]any{
				"trigger_type":     "contains",
				"trigger_value":    "warranty",
				"scope":            "all",
				"response_type":    "text",
				"response_content": "Our warranty covers 12 months.",
			},
		},
	}

	var toolResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", toolReq, &toolResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	rule, matched := h.MatchRule("Does this product have a warranty period?", false)
	require.True(t, matched)
	assert.Equal(t, "Our warranty covers 12 months.", rule.ResponseContent)
}

func TestTier3_Pairwise_GroupModeration_WelcomeTemplates(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":         "community@g.us",
		"anti_link_enabled": true,
		"welcome_enabled":   true,
		"welcome_template":  "Welcome @{name} to {group}!",
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	welcomeMsg := h.FormatWelcome("Welcome @{name} to {group}!", "Charlie", "Go Community")
	assert.Equal(t, "Welcome @Charlie to Go Community!", welcomeMsg)
	h.Dispatcher.SendText("community@g.us", welcomeMsg)
	assert.Len(t, h.Dispatcher.SentMessages, 1)

	modRes, violated := h.ModerateMessage("community@g.us", "charlie@s.whatsapp.net", "Join here: https://chat.whatsapp.com/XYZ123")
	require.True(t, violated)
	assert.True(t, modRes.ShouldRevoke)
}

func TestTier3_Pairwise_RESTAPI_AutoResponderRules(t *testing.T) {
	h := NewTestHarness(t)
	createPayload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    "catalog",
		"scope":            "all",
		"response_type":    "text",
		"response_content": "Catalog v1",
	}

	var createResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", createPayload, &createResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, code)
	resMap, ok := createResp.Results.(map[string]any)
	require.True(t, ok)
	ruleID := int64(resMap["id"].(float64))

	rule, matched := h.MatchRule("catalog", false)
	require.True(t, matched)
	assert.Equal(t, "Catalog v1", rule.ResponseContent)

	updatePayload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    "catalog",
		"scope":            "all",
		"response_type":    "text",
		"response_content": "Catalog v2 Updated",
	}
	var putResp APIResponse
	code, err = h.DoJSON(http.MethodPut, fmt.Sprintf("/bot/rules/%d", ruleID), updatePayload, &putResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	rule, matched = h.MatchRule("catalog", false)
	require.True(t, matched)
	assert.Equal(t, "Catalog v2 Updated", rule.ResponseContent)

	var delResp APIResponse
	code, err = h.DoJSON(http.MethodDelete, fmt.Sprintf("/bot/rules/%d", ruleID), nil, &delResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	_, matched = h.MatchRule("catalog", false)
	assert.False(t, matched)
}

func TestTier3_Pairwise_RESTAPI_AIConfig(t *testing.T) {
	h := NewTestHarness(t)
	newTemp := 0.3
	updatePayload := map[string]any{
		"model":       "gpt-4o",
		"temperature": newTemp,
	}
	var putResp APIResponse
	code, err := h.DoJSON(http.MethodPut, "/bot/ai/config", updatePayload, &putResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	h.SetOpenAIResponse("test pairwise ai", "AI response using updated model", nil, http.StatusOK)
	var chatResp APIResponse
	code, err = h.DoJSON(http.MethodPost, "/bot/ai/chat", ChatRequest{Message: "test pairwise ai"}, &chatResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := chatResp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "AI response using updated model", resMap["reply"])
	assert.Equal(t, "gpt-4o", resMap["model"])
}

func TestTier3_Pairwise_AutonomousTools_GroupModeration(t *testing.T) {
	h := NewTestHarness(t)
	_, _ = h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('secure_group@g.us', 1)`)

	_, violated := h.ModerateMessage("secure_group@g.us", "spammer@s.whatsapp.net", "Join https://chat.whatsapp.com/Forbidden")
	require.True(t, violated)

	toolReq := ToolRequest{
		Tool: "manage_group",
		Parameters: map[string]any{
			"action":       "remove",
			"group_jid":    "secure_group@g.us",
			"participants": []string{"spammer@s.whatsapp.net"},
		},
	}
	var toolResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", toolReq, &toolResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	assert.Len(t, h.Dispatcher.GroupActions, 1)
	assert.Equal(t, "remove", h.Dispatcher.GroupActions[0].Action)
	assert.Equal(t, "secure_group@g.us", h.Dispatcher.GroupActions[0].GroupJID)
	assert.Equal(t, []string{"spammer@s.whatsapp.net"}, h.Dispatcher.GroupActions[0].Participants)
}
