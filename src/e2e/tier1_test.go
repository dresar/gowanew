package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTier1_Feature1_PureGoSQLite_SchemaInitAndVersion(t *testing.T) {
	h := NewTestHarness(t)
	var version int
	err := h.DB.QueryRow(`SELECT version FROM bot_schema_info WHERE version = 1`).Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestTier1_Feature1_PureGoSQLite_AllTablesExist(t *testing.T) {
	h := NewTestHarness(t)
	requiredTables := []string{
		"bot_schema_info",
		"bot_rules",
		"bot_group_rules",
		"bot_ai_config",
		"bot_event_logs",
	}

	for _, table := range requiredTables {
		var name string
		err := h.DB.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		require.NoError(t, err, "table %s must exist", table)
		assert.Equal(t, table, name)
	}
}

func TestTier1_Feature1_PureGoSQLite_WALAndPragmasActive(t *testing.T) {
	h := NewTestHarness(t)
	var journalMode string
	err := h.DB.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", strings.ToLower(journalMode))

	var foreignKeys int
	err = h.DB.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys)
	require.NoError(t, err)
	assert.Equal(t, 1, foreignKeys)
}

func TestTier1_Feature1_PureGoSQLite_TableCheckConstraints(t *testing.T) {
	h := NewTestHarness(t)

	_, err := h.DB.Exec(`
		INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
		VALUES ('invalid_trigger', 'test', 'all', 'text', 'hello')
	`)
	require.Error(t, err)

	_, err = h.DB.Exec(`
		INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
		VALUES ('exact', 'test', 'invalid_scope', 'text', 'hello')
	`)
	require.Error(t, err)

	_, err = h.DB.Exec(`
		INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
		VALUES ('exact', 'test', 'all', 'invalid_response', 'hello')
	`)
	require.Error(t, err)
}

func TestTier1_Feature1_PureGoSQLite_ConcurrentReadWrite(t *testing.T) {
	h := NewTestHarness(t)
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := h.DB.Exec(
				`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', ?, 'all', 'text', 'resp')`,
				fmt.Sprintf("keyword_%d", idx),
			)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	var count int
	err := h.DB.QueryRow(`SELECT COUNT(*) FROM bot_rules`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestTier1_Feature2_AutoResponder_ExactTriggerMatch(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'ping', 'all', 'text', 'pong', 1)`)
	require.NoError(t, err)

	rule, matched := h.MatchRule("ping", false)
	require.True(t, matched)
	assert.Equal(t, "pong", rule.ResponseContent)

	_, matched = h.MatchRule("ping pong", false)
	assert.False(t, matched)
}

func TestTier1_Feature2_AutoResponder_ContainsTriggerMatch(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('contains', 'price', 'all', 'text', 'Pricing is $10', 1)`)
	require.NoError(t, err)

	rule, matched := h.MatchRule("What is your price list?", false)
	require.True(t, matched)
	assert.Equal(t, "Pricing is $10", rule.ResponseContent)

	_, matched = h.MatchRule("Hello there", false)
	assert.False(t, matched)
}

func TestTier1_Feature2_AutoResponder_StartsWithTriggerMatch(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('starts_with', '!help', 'all', 'text', 'Help Menu: 1. Rules 2. AI', 1)`)
	require.NoError(t, err)

	rule, matched := h.MatchRule("!help me please", false)
	require.True(t, matched)
	assert.Equal(t, "Help Menu: 1. Rules 2. AI", rule.ResponseContent)

	_, matched = h.MatchRule("please !help", false)
	assert.False(t, matched)
}

func TestTier1_Feature2_AutoResponder_RegexTriggerMatch(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('regex', '^order-[0-9]{4}$', 'all', 'text', 'Order lookup confirmed', 1)`)
	require.NoError(t, err)

	rule, matched := h.MatchRule("order-1234", false)
	require.True(t, matched)
	assert.Equal(t, "Order lookup confirmed", rule.ResponseContent)

	_, matched = h.MatchRule("order-abcd", false)
	assert.False(t, matched)
}

func TestTier1_Feature2_AutoResponder_ScopeFiltering(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'private_only', 'private', 'text', 'Secret', 1)`)
	require.NoError(t, err)
	_, err = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'group_only', 'group', 'text', 'Group announcement', 1)`)
	require.NoError(t, err)

	_, matched := h.MatchRule("private_only", true)
	assert.False(t, matched)

	rule, matched := h.MatchRule("private_only", false)
	require.True(t, matched)
	assert.Equal(t, "Secret", rule.ResponseContent)

	_, matched = h.MatchRule("group_only", false)
	assert.False(t, matched)

	rule, matched = h.MatchRule("group_only", true)
	require.True(t, matched)
	assert.Equal(t, "Group announcement", rule.ResponseContent)
}

func TestTier1_Feature3_GroupModeration_DetectInviteLink(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('group1@g.us', 1)`)
	require.NoError(t, err)

	res, violated := h.ModerateMessage("group1@g.us", "user1@s.whatsapp.net", "Join my group: https://chat.whatsapp.com/AbCdEf123")
	require.True(t, violated)
	assert.True(t, res.ShouldRevoke)
	assert.Equal(t, "anti_link_violation", res.Reason)
}

func TestTier1_Feature3_GroupModeration_DetectWaMeLink(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('group1@g.us', 1)`)
	require.NoError(t, err)

	res, violated := h.ModerateMessage("group1@g.us", "user1@s.whatsapp.net", "Contact me directly at wa.me/1234567890")
	require.True(t, violated)
	assert.True(t, res.ShouldRevoke)
}

func TestTier1_Feature3_GroupModeration_DetectGenericHttpLink(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('group1@g.us', 1)`)
	require.NoError(t, err)

	res, violated := h.ModerateMessage("group1@g.us", "user1@s.whatsapp.net", "Visit our store at https://example.com/shop")
	require.True(t, violated)
	assert.True(t, res.ShouldRevoke)
}

func TestTier1_Feature3_GroupModeration_RevokePayloadGenerated(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('modgroup@g.us', 1)`)
	require.NoError(t, err)

	res, violated := h.ModerateMessage("modgroup@g.us", "spammer@s.whatsapp.net", "spam link: http://spam.xyz")
	require.True(t, violated)
	if res.ShouldRevoke {
		h.Dispatcher.RevokeMessage("modgroup@g.us", "spammer@s.whatsapp.net", "MSG-101")
	}

	assert.Len(t, h.Dispatcher.Revocations, 1)
	assert.Equal(t, "modgroup@g.us", h.Dispatcher.Revocations[0].GroupJID)
	assert.Equal(t, "spammer@s.whatsapp.net", h.Dispatcher.Revocations[0].SenderJID)
	assert.Equal(t, "MSG-101", h.Dispatcher.Revocations[0].MessageID)
}

func TestTier1_Feature3_GroupModeration_DisabledAllowsLinks(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('open_group@g.us', 0)`)
	require.NoError(t, err)

	_, violated := h.ModerateMessage("open_group@g.us", "user@s.whatsapp.net", "https://chat.whatsapp.com/test")
	assert.False(t, violated)
}

func TestTier1_Feature4_WelcomeFarewell_FormatWelcomeNameGroup(t *testing.T) {
	h := NewTestHarness(t)
	template := "Welcome @{name} to {group}! Enjoy your stay."
	formatted := h.FormatWelcome(template, "Alice", "Golang Developers")
	assert.Equal(t, "Welcome @Alice to Golang Developers! Enjoy your stay.", formatted)
}

func TestTier1_Feature4_WelcomeFarewell_FormatFarewellNameGroup(t *testing.T) {
	h := NewTestHarness(t)
	template := "Goodbye @{name}, thanks for being part of {group}."
	formatted := h.FormatFarewell(template, "Bob", "Golang Developers")
	assert.Equal(t, "Goodbye @Bob, thanks for being part of Golang Developers.", formatted)
}

func TestTier1_Feature4_WelcomeFarewell_WelcomeDisabledNoOutput(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, welcome_enabled, welcome_template) VALUES ('group_nowelcome@g.us', 0, 'Welcome {name}')`)
	require.NoError(t, err)

	var welcomeEnabled int
	err = h.DB.QueryRow(`SELECT welcome_enabled FROM bot_group_rules WHERE group_jid = 'group_nowelcome@g.us'`).Scan(&welcomeEnabled)
	require.NoError(t, err)
	assert.Equal(t, 0, welcomeEnabled)
}

func TestTier1_Feature4_WelcomeFarewell_FarewellDisabledNoOutput(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, farewell_enabled, farewell_template) VALUES ('group_nofarewell@g.us', 0, 'Goodbye {name}')`)
	require.NoError(t, err)

	var farewellEnabled int
	err = h.DB.QueryRow(`SELECT farewell_enabled FROM bot_group_rules WHERE group_jid = 'group_nofarewell@g.us'`).Scan(&farewellEnabled)
	require.NoError(t, err)
	assert.Equal(t, 0, farewellEnabled)
}

func TestTier1_Feature4_WelcomeFarewell_SaveAndRetrieveTemplates(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":         "templategroup@g.us",
		"welcome_enabled":   true,
		"welcome_template":  "Hi {name} in {group}",
		"farewell_enabled":  true,
		"farewell_template": "Bye {name}",
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "SUCCESS", resp.Code)

	var getResp APIResponse
	code, err = h.DoJSON(http.MethodGet, "/bot/group-rules/templategroup@g.us", nil, &getResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resultsMap, ok := getResp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hi {name} in {group}", resultsMap["welcome_template"])
	assert.Equal(t, "Bye {name}", resultsMap["farewell_template"])
}

func TestTier1_Feature5_AIAssistant_DefaultSingletonConfig(t *testing.T) {
	h := NewTestHarness(t)
	var cfgResp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/ai/config", nil, &cfgResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "SUCCESS", cfgResp.Code)

	res, ok := cfgResp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "openai", res["provider"])
	assert.Equal(t, "!ai", res["trigger_prefix"])
}

func TestTier1_Feature5_AIAssistant_UpdateConfigFields(t *testing.T) {
	h := NewTestHarness(t)
	temp := 0.2
	updatePayload := map[string]any{
		"model":              "gpt-4o",
		"temperature":        temp,
		"trigger_prefix":     "!ask",
		"system_prompt":      "Act as an expert support engineer.",
		"auto_reply_enabled": true,
	}

	var putResp APIResponse
	code, err := h.DoJSON(http.MethodPut, "/bot/ai/config", updatePayload, &putResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var getResp APIResponse
	code, err = h.DoJSON(http.MethodGet, "/bot/ai/config", nil, &getResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	res, ok := getResp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", res["model"])
	assert.Equal(t, 0.2, res["temperature"])
	assert.Equal(t, "!ask", res["trigger_prefix"])
	assert.Equal(t, true, res["auto_reply_enabled"])
}

func TestTier1_Feature5_AIAssistant_PrefixTriggerRecognition(t *testing.T) {
	h := NewTestHarness(t)
	var prefix string
	err := h.DB.QueryRow(`SELECT trigger_prefix FROM bot_ai_config WHERE id = 1`).Scan(&prefix)
	require.NoError(t, err)

	incoming := "!ai What is the capital of Indonesia?"
	isAITrigger := strings.HasPrefix(incoming, prefix)
	assert.True(t, isAITrigger)
	cleanedPrompt := strings.TrimSpace(strings.TrimPrefix(incoming, prefix))
	assert.Equal(t, "What is the capital of Indonesia?", cleanedPrompt)
}

func TestTier1_Feature5_AIAssistant_IgnoreMessageWithoutPrefix(t *testing.T) {
	h := NewTestHarness(t)
	var prefix string
	var autoReply int
	err := h.DB.QueryRow(`SELECT trigger_prefix, auto_reply_enabled FROM bot_ai_config WHERE id = 1`).Scan(&prefix, &autoReply)
	require.NoError(t, err)

	incoming := "Regular message to chat"
	isAITrigger := autoReply == 1 || strings.HasPrefix(incoming, prefix)
	assert.False(t, isAITrigger)
}

func TestTier1_Feature5_AIAssistant_AutoReplyEnabledTriggersAll(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`UPDATE bot_ai_config SET auto_reply_enabled = 1 WHERE id = 1`)
	require.NoError(t, err)

	var prefix string
	var autoReply int
	err = h.DB.QueryRow(`SELECT trigger_prefix, auto_reply_enabled FROM bot_ai_config WHERE id = 1`).Scan(&prefix, &autoReply)
	require.NoError(t, err)

	incoming := "Any incoming text"
	isAITrigger := autoReply == 1 || strings.HasPrefix(incoming, prefix)
	assert.True(t, isAITrigger)
}

func TestTier1_Feature6_AutonomousTools_ListAvailableTools(t *testing.T) {
	h := NewTestHarness(t)
	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/ai/tools", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	tools, ok := resp.Results.([]any)
	require.True(t, ok)
	assert.Len(t, tools, 4)
}

func TestTier1_Feature6_AutonomousTools_ExecuteSendMessage(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "send_message",
		Parameters: map[string]any{
			"recipient": "12345@s.whatsapp.net",
			"message":   "Hello from autonomous tool",
		},
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Len(t, h.Dispatcher.SentMessages, 1)
	assert.Equal(t, "12345@s.whatsapp.net", h.Dispatcher.SentMessages[0].Recipient)
	assert.Equal(t, "Hello from autonomous tool", h.Dispatcher.SentMessages[0].Message)
}

func TestTier1_Feature6_AutonomousTools_ExecuteManageGroup(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "manage_group",
		Parameters: map[string]any{
			"action":       "remove",
			"group_jid":    "120363000@g.us",
			"participants": []string{"spammer1@s.whatsapp.net"},
		},
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Len(t, h.Dispatcher.GroupActions, 1)
	assert.Equal(t, "remove", h.Dispatcher.GroupActions[0].Action)
	assert.Equal(t, "120363000@g.us", h.Dispatcher.GroupActions[0].GroupJID)
}

func TestTier1_Feature6_AutonomousTools_ExecuteQueryChats(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "query_chats",
		Parameters: map[string]any{
			"action": "list_chats",
			"limit":  10,
		},
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "success", resMap["status"])
}

func TestTier1_Feature6_AutonomousTools_ExecuteUpdateRules(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "update_rules",
		Parameters: map[string]any{
			"action": "create_rule",
			"rule_data": map[string]any{
				"trigger_type":     "contains",
				"trigger_value":    "refund",
				"scope":            "all",
				"response_type":    "text",
				"response_content": "Please contact support for refund requests.",
			},
		},
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_rules WHERE trigger_value = 'refund'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTier1_Feature7_EventLogger_LogAutoReply(t *testing.T) {
	h := NewTestHarness(t)
	ruleID := int64(10)
	err := h.LogEvent(BotEventLog{
		EventType:       "auto_reply",
		RuleID:          &ruleID,
		SenderJID:       "user@s.whatsapp.net",
		GroupJID:        "",
		IncomingMessage: "hello",
		ResponseMessage: "hi there",
		LatencyMS:       45,
		Status:          "success",
	})
	require.NoError(t, err)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE event_type = 'auto_reply'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTier1_Feature7_EventLogger_LogGroupModeration(t *testing.T) {
	h := NewTestHarness(t)
	err := h.LogEvent(BotEventLog{
		EventType:       "group_moderation",
		SenderJID:       "spammer@s.whatsapp.net",
		GroupJID:        "group@g.us",
		IncomingMessage: "https://chat.whatsapp.com/test",
		ResponseMessage: "Message revoked due to anti-link policy",
		LatencyMS:       12,
		Status:          "success",
	})
	require.NoError(t, err)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE event_type = 'group_moderation'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTier1_Feature7_EventLogger_LogAIChat(t *testing.T) {
	h := NewTestHarness(t)
	err := h.LogEvent(BotEventLog{
		EventType:       "ai_chat",
		SenderJID:       "client@s.whatsapp.net",
		IncomingMessage: "!ai What is Go?",
		ResponseMessage: "Go is an open source programming language.",
		LatencyMS:       680,
		Status:          "success",
	})
	require.NoError(t, err)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE event_type = 'ai_chat'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTier1_Feature7_EventLogger_LogAITool(t *testing.T) {
	h := NewTestHarness(t)
	err := h.LogEvent(BotEventLog{
		EventType:       "ai_tool",
		SenderJID:       "ai-engine",
		IncomingMessage: "send_message",
		ResponseMessage: "MOCK-MSG-1",
		LatencyMS:       50,
		Status:          "success",
	})
	require.NoError(t, err)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs WHERE event_type = 'ai_tool'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTier1_Feature7_EventLogger_FilterLogsByEventTypeAndStatus(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success", LatencyMS: 10})
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u2", Status: "failed", LatencyMS: 15})
	_ = h.LogEvent(BotEventLog{EventType: "group_moderation", SenderJID: "u3", Status: "success", LatencyMS: 20})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?event_type=auto_reply&status=success", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), resMap["total"])
}

func TestTier1_Feature8_RESTAPI_CreateAndListRules(t *testing.T) {
	h := NewTestHarness(t)
	rulePayload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    "hello",
		"scope":            "all",
		"response_type":    "text",
		"response_content": "world",
	}

	var createResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", rulePayload, &createResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, code)

	var listResp APIResponse
	code, err = h.DoJSON(http.MethodGet, "/bot/rules", nil, &listResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	rulesList, ok := listResp.Results.([]any)
	require.True(t, ok)
	assert.Len(t, rulesList, 1)
}

func TestTier1_Feature8_RESTAPI_GetAndUpdateRuleByID(t *testing.T) {
	h := NewTestHarness(t)
	res, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 'key1', 'all', 'text', 'val1')`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	var getResp APIResponse
	code, err := h.DoJSON(http.MethodGet, fmt.Sprintf("/bot/rules/%d", id), nil, &getResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	updatePayload := map[string]any{
		"trigger_type":     "contains",
		"trigger_value":    "key1_updated",
		"scope":            "private",
		"response_type":    "text",
		"response_content": "val1_updated",
	}
	var putResp APIResponse
	code, err = h.DoJSON(http.MethodPut, fmt.Sprintf("/bot/rules/%d", id), updatePayload, &putResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var verifyVal string
	err = h.DB.QueryRow(`SELECT response_content FROM bot_rules WHERE id = ?`, id).Scan(&verifyVal)
	require.NoError(t, err)
	assert.Equal(t, "val1_updated", verifyVal)
}

func TestTier1_Feature8_RESTAPI_ToggleRuleStatus(t *testing.T) {
	h := NewTestHarness(t)
	res, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'toggle_me', 'all', 'text', 'ok', 1)`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	var patchResp APIResponse
	code, err := h.DoJSON(http.MethodPatch, fmt.Sprintf("/bot/rules/%d/toggle", id), nil, &patchResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var isActive int
	err = h.DB.QueryRow(`SELECT is_active FROM bot_rules WHERE id = ?`, id).Scan(&isActive)
	require.NoError(t, err)
	assert.Equal(t, 0, isActive)

	code, err = h.DoJSON(http.MethodPatch, fmt.Sprintf("/bot/rules/%d/toggle", id), nil, &patchResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	err = h.DB.QueryRow(`SELECT is_active FROM bot_rules WHERE id = ?`, id).Scan(&isActive)
	require.NoError(t, err)
	assert.Equal(t, 1, isActive)
}

func TestTier1_Feature8_RESTAPI_DeleteRuleByID(t *testing.T) {
	h := NewTestHarness(t)
	res, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 'delete_me', 'all', 'text', 'bye')`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	var delResp APIResponse
	code, err := h.DoJSON(http.MethodDelete, fmt.Sprintf("/bot/rules/%d", id), nil, &delResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_rules WHERE id = ?`, id).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTier1_Feature8_RESTAPI_GroupRulesCRUD(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":         "crud_group@g.us",
		"anti_link_enabled": true,
	}

	var postResp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &postResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var getResp APIResponse
	code, err = h.DoJSON(http.MethodGet, "/bot/group-rules/crud_group@g.us", nil, &getResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var delResp APIResponse
	code, err = h.DoJSON(http.MethodDelete, "/bot/group-rules/crud_group@g.us", nil, &delResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier1_Feature9_SidebarNav_HTMLContainsBotNavigationGroup(t *testing.T) {
	h := NewTestHarness(t)
	resp, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "Bot & Automation")
}

func TestTier1_Feature9_SidebarNav_HTMLContainsAutoResponderLink(t *testing.T) {
	h := NewTestHarness(t)
	_, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Contains(t, body, "Auto Responder")
}

func TestTier1_Feature9_SidebarNav_HTMLContainsAIAssistantLink(t *testing.T) {
	h := NewTestHarness(t)
	_, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Contains(t, body, "AI Assistant")
}

func TestTier1_Feature9_SidebarNav_HTMLContainsGroupModerationLink(t *testing.T) {
	h := NewTestHarness(t)
	_, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Contains(t, body, "Group Moderation")
}

func TestTier1_Feature9_SidebarNav_HTMLContainsBotLogsLink(t *testing.T) {
	h := NewTestHarness(t)
	_, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Contains(t, body, "Bot Activity Logs")
}

func TestTier1_Feature10_AutoResponderUI_ListActiveFilter(t *testing.T) {
	h := NewTestHarness(t)
	_, _ = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'active_kw', 'all', 'text', 'res', 1)`)
	_, _ = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'inactive_kw', 'all', 'text', 'res', 0)`)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/rules?active=true", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	list, ok := resp.Results.([]any)
	require.True(t, ok)
	assert.Len(t, list, 1)
}

func TestTier1_Feature10_AutoResponderUI_ListScopeFilter(t *testing.T) {
	h := NewTestHarness(t)
	_, _ = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 's_priv', 'private', 'text', 'res')`)
	_, _ = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 's_grp', 'group', 'text', 'res')`)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/rules?scope=private", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	list, ok := resp.Results.([]any)
	require.True(t, ok)
	assert.Len(t, list, 1)
}

func TestTier1_Feature10_AutoResponderUI_SearchFilter(t *testing.T) {
	h := NewTestHarness(t)
	_, _ = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 'alpha_term', 'all', 'text', 'alpha response')`)
	_, _ = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 'beta_term', 'all', 'text', 'beta response')`)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/rules?search=alpha", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	list, ok := resp.Results.([]any)
	require.True(t, ok)
	assert.Len(t, list, 1)
}

func TestTier1_Feature10_AutoResponderUI_MediaResponseFields(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"trigger_type":     "contains",
		"trigger_value":    "brochure",
		"scope":            "all",
		"response_type":    "media",
		"response_content": "Here is our product brochure",
		"media_url":        "https://example.com/brochure.pdf",
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "media", resMap["response_type"])
	assert.Equal(t, "https://example.com/brochure.pdf", resMap["media_url"])
}

func TestTier1_Feature10_AutoResponderUI_RequiredFieldValidation(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"trigger_type":  "exact",
		"trigger_value": "",
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "BAD_REQUEST", resp.Code)
}

func TestTier1_Feature11_AIConsole_InteractiveChatPostSuccess(t *testing.T) {
	h := NewTestHarness(t)
	h.SetOpenAIResponse("test chat message", "Hello, I am testing the AI assistant.", nil, http.StatusOK)

	payload := ChatRequest{
		Message: "test chat message",
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello, I am testing the AI assistant.", resMap["reply"])
}

func TestTier1_Feature11_AIConsole_ChatHistoryContext(t *testing.T) {
	h := NewTestHarness(t)
	h.SetOpenAIResponse("what was my name?", "Your name is Alice.", nil, http.StatusOK)

	payload := ChatRequest{
		Message: "what was my name?",
		ChatHistory: []ChatMessage{
			{Role: "user", Content: "My name is Alice."},
			{Role: "assistant", Content: "Nice to meet you Alice."},
		},
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Your name is Alice.", resMap["reply"])
}

func TestTier1_Feature11_AIConsole_CustomTemperatureParam(t *testing.T) {
	h := NewTestHarness(t)
	temp := 0.9
	payload := ChatRequest{
		Message:     "creative idea",
		Temperature: &temp,
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier1_Feature11_AIConsole_CustomModelParam(t *testing.T) {
	h := NewTestHarness(t)
	payload := ChatRequest{
		Message: "code review",
		Model:   "gpt-4o",
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", resMap["model"])
}

func TestTier1_Feature11_AIConsole_ToolsListDisplay(t *testing.T) {
	h := NewTestHarness(t)
	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/ai/tools", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	tools, ok := resp.Results.([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(tools), 4)
}

func TestTier1_Feature12_GroupModerationUI_AntiLinkToggleUpdate(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":         "toggle_link@g.us",
		"anti_link_enabled": true,
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var antiLink int
	err = h.DB.QueryRow(`SELECT anti_link_enabled FROM bot_group_rules WHERE group_jid = 'toggle_link@g.us'`).Scan(&antiLink)
	require.NoError(t, err)
	assert.Equal(t, 1, antiLink)
}

func TestTier1_Feature12_GroupModerationUI_WelcomeTemplateUpdate(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":        "welcome_update@g.us",
		"welcome_enabled":  true,
		"welcome_template": "Welcome to our group, {name}!",
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var template string
	err = h.DB.QueryRow(`SELECT welcome_template FROM bot_group_rules WHERE group_jid = 'welcome_update@g.us'`).Scan(&template)
	require.NoError(t, err)
	assert.Equal(t, "Welcome to our group, {name}!", template)
}

func TestTier1_Feature12_GroupModerationUI_FarewellTemplateUpdate(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":         "farewell_update@g.us",
		"farewell_enabled":  true,
		"farewell_template": "So long, {name} from {group}.",
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var template string
	err = h.DB.QueryRow(`SELECT farewell_template FROM bot_group_rules WHERE group_jid = 'farewell_update@g.us'`).Scan(&template)
	require.NoError(t, err)
	assert.Equal(t, "So long, {name} from {group}.", template)
}

func TestTier1_Feature12_GroupModerationUI_FetchByGroupJID(t *testing.T) {
	h := NewTestHarness(t)
	_, _ = h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('fetch_target@g.us', 1)`)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/group-rules/fetch_target@g.us", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "fetch_target@g.us", resMap["group_jid"])
}

func TestTier1_Feature12_GroupModerationUI_DeleteGroupRule(t *testing.T) {
	h := NewTestHarness(t)
	_, _ = h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('del_target@g.us', 1)`)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodDelete, "/bot/group-rules/del_target@g.us", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_group_rules WHERE group_jid = 'del_target@g.us'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTier1_Feature13_LogsUI_PaginationLimitOffset(t *testing.T) {
	h := NewTestHarness(t)
	for i := 0; i < 15; i++ {
		_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: fmt.Sprintf("user_%d", i), LatencyMS: 5, Status: "success"})
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?limit=5&offset=5", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(15), resMap["total"])
	logsList, ok := resMap["logs"].([]any)
	require.True(t, ok)
	assert.Len(t, logsList, 5)
}

func TestTier1_Feature13_LogsUI_FilterByEventType(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success"})
	_ = h.LogEvent(BotEventLog{EventType: "ai_chat", SenderJID: "u2", Status: "success"})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?event_type=ai_chat", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), resMap["total"])
}

func TestTier1_Feature13_LogsUI_FilterByStatus(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success"})
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u2", Status: "failed"})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?status=failed", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), resMap["total"])
}

func TestTier1_Feature13_LogsUI_FilterByGroupJID(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "group_moderation", SenderJID: "u1", GroupJID: "grp_a@g.us", Status: "success"})
	_ = h.LogEvent(BotEventLog{EventType: "group_moderation", SenderJID: "u2", GroupJID: "grp_b@g.us", Status: "success"})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?group_jid=grp_a@g.us", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), resMap["total"])
}

func TestTier1_Feature13_LogsUI_ClearLogsDelete(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success"})
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u2", Status: "success"})

	var delResp APIResponse
	code, err := h.DoJSON(http.MethodDelete, "/bot/logs", nil, &delResp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_event_logs`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTier1_Feature14_SingleFileBuild_RootServesHTML(t *testing.T) {
	h := NewTestHarness(t)
	resp, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "<!DOCTYPE html>")
}

func TestTier1_Feature14_SingleFileBuild_ContentTypeUTF8(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	contentType := resp.Header.Get("Content-Type")
	assert.Contains(t, strings.ToLower(contentType), "text/html")
	assert.Contains(t, strings.ToLower(contentType), "utf-8")
}

func TestTier1_Feature14_SingleFileBuild_AppRootDivPresent(t *testing.T) {
	h := NewTestHarness(t)
	_, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Contains(t, body, `<div id="root">`)
}

func TestTier1_Feature14_SingleFileBuild_StaticsEndpointReachable(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodGet, "/statics/test.png", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTier1_Feature14_SingleFileBuild_CORSHeadersPresent(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodOptions, "/bot/rules", nil)
	require.NoError(t, err)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}
