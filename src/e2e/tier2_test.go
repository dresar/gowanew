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

func TestTier2_Boundary_Feature1_SQLInjectionInTrigger(t *testing.T) {
	h := NewTestHarness(t)
	injectionPayload := "'; DROP TABLE bot_rules; --"
	res, err := h.DB.Exec(
		`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', ?, 'all', 'text', 'safe')`,
		injectionPayload,
	)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_rules WHERE id = ?`, id).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var tableCount int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'bot_rules'`).Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 1, tableCount)
}

func TestTier2_Boundary_Feature1_TriggerTypeCheckViolation(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('wildcard', 'val', 'all', 'text', 'res')`)
	require.Error(t, err)
}

func TestTier2_Boundary_Feature1_ScopeCheckViolation(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 'val', 'broadcast', 'text', 'res')`)
	require.Error(t, err)
}

func TestTier2_Boundary_Feature1_IsActiveCheckViolation(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'val', 'all', 'text', 'res', 2)`)
	require.Error(t, err)

	_, err = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'val', 'all', 'text', 'res', -1)`)
	require.Error(t, err)
}

func TestTier2_Boundary_Feature1_LargePayloadStorage(t *testing.T) {
	h := NewTestHarness(t)
	largeText := strings.Repeat("A", 100*1024)
	res, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', 'large', 'all', 'text', ?)`, largeText)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	var retrieved string
	err = h.DB.QueryRow(`SELECT response_content FROM bot_rules WHERE id = ?`, id).Scan(&retrieved)
	require.NoError(t, err)
	assert.Equal(t, len(largeText), len(retrieved))
}

func TestTier2_Boundary_Feature2_EmptyTriggerRejected(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    "",
		"response_content": "val",
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestTier2_Boundary_Feature2_CaseInsensitiveMatching(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'hELlo', 'all', 'text', 'world', 1)`)
	require.NoError(t, err)

	rule, matched := h.MatchRule("Hello", false)
	require.True(t, matched)
	assert.Equal(t, "world", rule.ResponseContent)

	rule, matched = h.MatchRule("HELLO", false)
	require.True(t, matched)
	assert.Equal(t, "world", rule.ResponseContent)
}

func TestTier2_Boundary_Feature2_UnicodeAndEmojiTriggers(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'halo 👋 selamat pagi', 'all', 'text', 'pagi juga! ☀️', 1)`)
	require.NoError(t, err)

	rule, matched := h.MatchRule("halo 👋 selamat pagi", false)
	require.True(t, matched)
	assert.Equal(t, "pagi juga! ☀️", rule.ResponseContent)
}

func TestTier2_Boundary_Feature2_InvalidRegexSyntaxHandling(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('regex', '[a-z', 'all', 'text', 'bad regex', 1)`)
	require.NoError(t, err)

	_, matched := h.MatchRule("test", false)
	assert.False(t, matched)
}

func TestTier2_Boundary_Feature2_InactiveRuleNeverTriggers(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'match_me', 'all', 'text', 'never', 0)`)
	require.NoError(t, err)

	_, matched := h.MatchRule("match_me", false)
	assert.False(t, matched)
}

func TestTier2_Boundary_Feature3_ObfuscatedUrlsDetection(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('mod1@g.us', 1)`)
	require.NoError(t, err)

	res, violated := h.ModerateMessage("mod1@g.us", "u1@s.whatsapp.net", "Join wa.me/62812345678")
	require.True(t, violated)
	assert.True(t, res.ShouldRevoke)
}

func TestTier2_Boundary_Feature3_MultipleUrlsInSingleMessage(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('mod2@g.us', 1)`)
	require.NoError(t, err)

	text := "Check https://github.com and join https://chat.whatsapp.com/Room123"
	res, violated := h.ModerateMessage("mod2@g.us", "u1@s.whatsapp.net", text)
	require.True(t, violated)
	assert.True(t, res.ShouldRevoke)
}

func TestTier2_Boundary_Feature3_EmptyMessageNoPanic(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('mod3@g.us', 1)`)
	require.NoError(t, err)

	res, violated := h.ModerateMessage("mod3@g.us", "u1@s.whatsapp.net", "")
	assert.False(t, violated)
	assert.Nil(t, res)
}

func TestTier2_Boundary_Feature3_LIDJIDNormalization(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('mod4@g.us', 1)`)
	require.NoError(t, err)

	lidSender := "987654321@lid"
	res, violated := h.ModerateMessage("mod4@g.us", lidSender, "https://chat.whatsapp.com/test")
	require.True(t, violated)
	assert.True(t, res.ShouldRevoke)

	normalizedSender := strings.Replace(lidSender, "@lid", "@s.whatsapp.net", 1)
	h.Dispatcher.RevokeMessage("mod4@g.us", normalizedSender, "REVOKE-1")
	assert.Equal(t, "987654321@s.whatsapp.net", h.Dispatcher.Revocations[0].SenderJID)
}

func TestTier2_Boundary_Feature3_NonGroupJIDIgnored(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`INSERT INTO bot_group_rules (group_jid, anti_link_enabled) VALUES ('group@g.us', 1)`)
	require.NoError(t, err)

	_, violated := h.ModerateMessage("private_chat@s.whatsapp.net", "u1@s.whatsapp.net", "https://chat.whatsapp.com/test")
	assert.False(t, violated)
}

func TestTier2_Boundary_Feature4_EmptyTemplateNoMessage(t *testing.T) {
	h := NewTestHarness(t)
	res := h.FormatWelcome("", "Alice", "Devs")
	assert.Empty(t, res)
}

func TestTier2_Boundary_Feature4_TemplateMissingPlaceholders(t *testing.T) {
	h := NewTestHarness(t)
	res := h.FormatWelcome("Welcome new member!", "Alice", "Devs")
	assert.Equal(t, "Welcome new member!", res)
}

func TestTier2_Boundary_Feature4_SpecialCharsAndEmojisInName(t *testing.T) {
	h := NewTestHarness(t)
	specialName := `<script>alert("xss")</script> 🚀`
	res := h.FormatWelcome("Hello {name} in {group}", specialName, "Devs")
	assert.Equal(t, `Hello <script>alert("xss")</script> 🚀 in Devs`, res)
}

func TestTier2_Boundary_Feature4_ExtremelyLongNameAndGroup(t *testing.T) {
	h := NewTestHarness(t)
	longName := strings.Repeat("N", 500)
	longGroup := strings.Repeat("G", 500)
	res := h.FormatWelcome("{name} joined {group}", longName, longGroup)
	assert.Equal(t, 1008, len(res))
}

func TestTier2_Boundary_Feature4_MultipleReplacementsInTemplate(t *testing.T) {
	h := NewTestHarness(t)
	template := "{name}, welcome to {group}! Hey {name}, check pinned msg in {group}."
	res := h.FormatWelcome(template, "Bob", "Golang")
	assert.Equal(t, "Bob, welcome to Golang! Hey Bob, check pinned msg in Golang.", res)
}

func TestTier2_Boundary_Feature5_TemperatureBounds0And2(t *testing.T) {
	h := NewTestHarness(t)
	t0 := 0.0
	payload0 := map[string]any{"temperature": t0}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPut, "/bot/ai/config", payload0, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	t2 := 2.0
	payload2 := map[string]any{"temperature": t2}
	code, err = h.DoJSON(http.MethodPut, "/bot/ai/config", payload2, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var tempInDB float64
	err = h.DB.QueryRow(`SELECT temperature FROM bot_ai_config WHERE id = 1`).Scan(&tempInDB)
	require.NoError(t, err)
	assert.Equal(t, 2.0, tempInDB)
}

func TestTier2_Boundary_Feature5_EmptyAPIKeyErrorMessage(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{"api_key": "invalid-token"}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPut, "/bot/ai/config", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	chatPayload := ChatRequest{Message: "hello"}
	code, err = h.DoJSON(http.MethodPost, "/bot/ai/chat", chatPayload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, code)
	assert.Equal(t, "AI_ERROR", resp.Code)
}

func TestTier2_Boundary_Feature5_HugeSystemPrompt(t *testing.T) {
	h := NewTestHarness(t)
	hugePrompt := strings.Repeat("Rule: Be precise. ", 500)
	payload := map[string]any{"system_prompt": hugePrompt}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPut, "/bot/ai/config", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var retrieved string
	err = h.DB.QueryRow(`SELECT system_prompt FROM bot_ai_config WHERE id = 1`).Scan(&retrieved)
	require.NoError(t, err)
	assert.Equal(t, hugePrompt, retrieved)
}

func TestTier2_Boundary_Feature5_PrefixWithTrailingSpaces(t *testing.T) {
	h := NewTestHarness(t)
	_, err := h.DB.Exec(`UPDATE bot_ai_config SET trigger_prefix = '!ai ' WHERE id = 1`)
	require.NoError(t, err)

	var prefix string
	err = h.DB.QueryRow(`SELECT trigger_prefix FROM bot_ai_config WHERE id = 1`).Scan(&prefix)
	require.NoError(t, err)

	incoming := "!ai    summarize this document"
	isMatch := strings.HasPrefix(incoming, strings.TrimSpace(prefix))
	assert.True(t, isMatch)
	query := strings.TrimSpace(strings.TrimPrefix(incoming, strings.TrimSpace(prefix)))
	assert.Equal(t, "summarize this document", query)
}

func TestTier2_Boundary_Feature5_ConcurrentAIRequests(t *testing.T) {
	h := NewTestHarness(t)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var resp APIResponse
			code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", ChatRequest{Message: fmt.Sprintf("req_%d", idx)}, &resp)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, code)
		}(i)
	}
	wg.Wait()
}

func TestTier2_Boundary_Feature6_SendMessageMissingRecipient(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "send_message",
		Parameters: map[string]any{
			"message": "hello without recipient",
		},
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, "TOOL_ERROR", resp.Code)
}

func TestTier2_Boundary_Feature6_ManageGroupInvalidAction(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "manage_group",
		Parameters: map[string]any{
			"action":    "destroy_group",
			"group_jid": "group@g.us",
		},
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestTier2_Boundary_Feature6_MalformedJSONPayload(t *testing.T) {
	h := NewTestHarness(t)
	resp, body, err := h.DoRequest(http.MethodPost, "/bot/ai/tools", "malformed json")
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, "BAD_REQUEST")
}

func TestTier2_Boundary_Feature6_UpdateRulesInvalidTriggerType(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "update_rules",
		Parameters: map[string]any{
			"action": "create_rule",
			"rule_data": map[string]any{
				"trigger_type":     "invalid_type",
				"trigger_value":    "bad",
				"response_content": "bad",
			},
		},
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestTier2_Boundary_Feature6_QueryChatsNegativeLimitOffset(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "query_chats",
		Parameters: map[string]any{
			"action": "list_chats",
			"limit":  -5,
			"offset": -10,
		},
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier2_Boundary_Feature7_HugeLogMessageBody(t *testing.T) {
	h := NewTestHarness(t)
	hugeIncoming := strings.Repeat("M", 50*1024)
	err := h.LogEvent(BotEventLog{
		EventType:       "auto_reply",
		SenderJID:       "user@s.whatsapp.net",
		IncomingMessage: hugeIncoming,
		ResponseMessage: "ok",
		Status:          "success",
	})
	require.NoError(t, err)

	var length int
	err = h.DB.QueryRow(`SELECT LENGTH(incoming_message) FROM bot_event_logs WHERE event_type = 'auto_reply'`).Scan(&length)
	require.NoError(t, err)
	assert.Equal(t, len(hugeIncoming), length)
}

func TestTier2_Boundary_Feature7_ZeroAndHighLatencyTracking(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", LatencyMS: 0, Status: "success"})
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u2", LatencyMS: 60000, Status: "success"})

	var minLatency, maxLatency int64
	err := h.DB.QueryRow(`SELECT MIN(latency_ms), MAX(latency_ms) FROM bot_event_logs`).Scan(&minLatency, &maxLatency)
	require.NoError(t, err)
	assert.Equal(t, int64(0), minLatency)
	assert.Equal(t, int64(60000), maxLatency)
}

func TestTier2_Boundary_Feature7_InvalidEventTypeRejected(t *testing.T) {
	h := NewTestHarness(t)
	err := h.LogEvent(BotEventLog{
		EventType: "invalid_event",
		SenderJID: "u1",
		Status:    "success",
	})
	require.Error(t, err)
}

func TestTier2_Boundary_Feature7_NegativeLimitSanitization(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success"})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?limit=-10&offset=-5", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(50), resMap["limit"])
	assert.Equal(t, float64(0), resMap["offset"])
}

func TestTier2_Boundary_Feature7_ClearLogsOnEmptyDB(t *testing.T) {
	h := NewTestHarness(t)
	var resp APIResponse
	code, err := h.DoJSON(http.MethodDelete, "/bot/logs", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0), resMap["deleted_count"])
}

func TestTier2_Boundary_Feature8_GetNonExistentRule404(t *testing.T) {
	h := NewTestHarness(t)
	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/rules/99999", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, code)
	assert.Equal(t, "NOT_FOUND", resp.Code)
}

func TestTier2_Boundary_Feature8_NonNumericRuleID400(t *testing.T) {
	h := NewTestHarness(t)
	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/rules/invalid_id", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestTier2_Boundary_Feature8_PutEmptyBody400(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodPut, "/bot/rules/1", "invalid json")
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTier2_Boundary_Feature8_DuplicateGroupJIDUpsert(t *testing.T) {
	h := NewTestHarness(t)
	payload1 := map[string]any{"group_jid": "dup@g.us", "anti_link_enabled": true}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload1, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	payload2 := map[string]any{"group_jid": "dup@g.us", "anti_link_enabled": false}
	code, err = h.DoJSON(http.MethodPost, "/bot/group-rules", payload2, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var count int
	err = h.DB.QueryRow(`SELECT COUNT(*) FROM bot_group_rules WHERE group_jid = 'dup@g.us'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTier2_Boundary_Feature8_MethodNotAllowed(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodPatch, "/bot/rules", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestTier2_Boundary_Feature9_MobileViewportRender(t *testing.T) {
	h := NewTestHarness(t)
	req, _ := http.NewRequest(http.MethodGet, h.BaseURL+"/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)")
	resp, err := h.HTTPClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTier2_Boundary_Feature9_ActiveLinkIndicator(t *testing.T) {
	h := NewTestHarness(t)
	_, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Contains(t, body, `href="#rules"`)
	assert.Contains(t, body, `href="#ai"`)
}

func TestTier2_Boundary_Feature9_DeepLinkPreservation(t *testing.T) {
	h := NewTestHarness(t)
	resp, body, err := h.DoRequest(http.MethodGet, "/#rules", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, `<div id="root">`)
}

func TestTier2_Boundary_Feature9_RootRouteTrailingSlash(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTier2_Boundary_Feature9_NoDuplicateNavItems(t *testing.T) {
	h := NewTestHarness(t)
	_, body, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	count := strings.Count(body, "Auto Responder")
	assert.Equal(t, 1, count)
}

func TestTier2_Boundary_Feature10_MaxTriggerValueLength(t *testing.T) {
	h := NewTestHarness(t)
	maxStr := strings.Repeat("k", 255)
	payload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    maxStr,
		"response_content": "val",
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, code)
}

func TestTier2_Boundary_Feature10_RapidToggleTransitions(t *testing.T) {
	h := NewTestHarness(t)
	res, err := h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES ('exact', 'rapid', 'all', 'text', 'ok', 1)`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()

	for i := 0; i < 6; i++ {
		var resp APIResponse
		code, err := h.DoJSON(http.MethodPatch, fmt.Sprintf("/bot/rules/%d/toggle", id), nil, &resp)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
	}

	var isActive int
	err = h.DB.QueryRow(`SELECT is_active FROM bot_rules WHERE id = ?`, id).Scan(&isActive)
	require.NoError(t, err)
	assert.Equal(t, 1, isActive)
}

func TestTier2_Boundary_Feature10_SearchFilterSpecialRegexChars(t *testing.T) {
	h := NewTestHarness(t)
	special := `.*+?^${}()|[]\`
	_, _ = h.DB.Exec(`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content) VALUES ('exact', ?, 'all', 'text', 'res')`, special)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/rules?search="+special, nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier2_Boundary_Feature10_EmptyResponseContentRejected(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    "valid_key",
		"response_content": "",
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestTier2_Boundary_Feature10_MediaURLSchemeValidation(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"trigger_type":     "exact",
		"trigger_value":    "media_key",
		"response_type":    "media",
		"response_content": "caption",
		"media_url":        "ftp://invalid.com/pic.jpg",
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, code)
}

func TestTier2_Boundary_Feature11_InteractiveChatEmptyMessage(t *testing.T) {
	h := NewTestHarness(t)
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", ChatRequest{Message: ""}, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestTier2_Boundary_Feature11_ChatHistoryLargeContext(t *testing.T) {
	h := NewTestHarness(t)
	history := make([]ChatMessage, 0, 20)
	for i := 0; i < 20; i++ {
		history = append(history, ChatMessage{Role: "user", Content: fmt.Sprintf("Question %d", i)})
		history = append(history, ChatMessage{Role: "assistant", Content: fmt.Sprintf("Answer %d", i)})
	}

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", ChatRequest{Message: "Summarize conversation", ChatHistory: history}, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier2_Boundary_Feature11_OpenAIUnauthorizedError(t *testing.T) {
	h := NewTestHarness(t)
	_, _ = h.DB.Exec(`UPDATE bot_ai_config SET api_key = 'invalid-token' WHERE id = 1`)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", ChatRequest{Message: "hi"}, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, code)
}

func TestTier2_Boundary_Feature11_NestedToolParameters(t *testing.T) {
	h := NewTestHarness(t)
	req := ToolRequest{
		Tool: "update_rules",
		Parameters: map[string]any{
			"action": "create_rule",
			"rule_data": map[string]any{
				"trigger_type":     "exact",
				"trigger_value":    "nested_key",
				"response_content": "nested_val",
			},
		},
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/tools", req, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier2_Boundary_Feature11_XSSSanitizationInOutput(t *testing.T) {
	h := NewTestHarness(t)
	h.SetOpenAIResponse("test xss", `<script>alert(1)</script>`, nil, http.StatusOK)

	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/ai/chat", ChatRequest{Message: "test xss"}, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, resMap["reply"], `<script>alert(1)</script>`)
}

func TestTier2_Boundary_Feature12_TemplateExceeding1000Chars(t *testing.T) {
	h := NewTestHarness(t)
	hugeTemplate := strings.Repeat("Welcome {name}! ", 100)
	payload := map[string]any{
		"group_jid":        "huge_tpl@g.us",
		"welcome_template": hugeTemplate,
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier2_Boundary_Feature12_InvalidGroupJIDRejected(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid": "",
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestTier2_Boundary_Feature12_BothTemplatesDisabledAntiLinkActive(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":         "antilink_only@g.us",
		"anti_link_enabled": true,
		"welcome_enabled":   false,
		"farewell_enabled":  false,
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	res, violated := h.ModerateMessage("antilink_only@g.us", "u@s.whatsapp.net", "http://spam.org")
	require.True(t, violated)
	assert.True(t, res.ShouldRevoke)
}

func TestTier2_Boundary_Feature12_AntiLinkToggleWithoutTemplates(t *testing.T) {
	h := NewTestHarness(t)
	payload := map[string]any{
		"group_jid":         "bare_toggle@g.us",
		"anti_link_enabled": true,
	}
	var resp APIResponse
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var wTemplate string
	err = h.DB.QueryRow(`SELECT welcome_template FROM bot_group_rules WHERE group_jid = 'bare_toggle@g.us'`).Scan(&wTemplate)
	require.NoError(t, err)
	assert.Empty(t, wTemplate)
}

func TestTier2_Boundary_Feature12_ResetTemplatesToEmpty(t *testing.T) {
	h := NewTestHarness(t)
	payload1 := map[string]any{"group_jid": "reset_grp@g.us", "welcome_template": "hello {name}"}
	var resp APIResponse
	_, _ = h.DoJSON(http.MethodPost, "/bot/group-rules", payload1, &resp)

	payload2 := map[string]any{"group_jid": "reset_grp@g.us", "welcome_template": ""}
	code, err := h.DoJSON(http.MethodPost, "/bot/group-rules", payload2, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	var wTemplate string
	err = h.DB.QueryRow(`SELECT welcome_template FROM bot_group_rules WHERE group_jid = 'reset_grp@g.us'`).Scan(&wTemplate)
	require.NoError(t, err)
	assert.Empty(t, wTemplate)
}

func TestTier2_Boundary_Feature13_NonExistentStatusEmptyList(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success"})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?status=non_existent_status", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0), resMap["total"])
}

func TestTier2_Boundary_Feature13_OffsetExceedingTotalCount(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success"})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?offset=1000", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	logsList, ok := resMap["logs"].([]any)
	require.True(t, ok)
	assert.Empty(t, logsList)
}

func TestTier2_Boundary_Feature13_SearchWithSQLWildcards(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", IncomingMessage: "100% discount", Status: "success"})

	var resp APIResponse
	code, err := h.DoJSON(http.MethodGet, "/bot/logs?search=%", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
}

func TestTier2_Boundary_Feature13_RapidClearAndQuery(t *testing.T) {
	h := NewTestHarness(t)
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", Status: "success"})

	var resp APIResponse
	_, _ = h.DoJSON(http.MethodDelete, "/bot/logs", nil, &resp)
	code, err := h.DoJSON(http.MethodGet, "/bot/logs", nil, &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)
	resMap, ok := resp.Results.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0), resMap["total"])
}

func TestTier2_Boundary_Feature13_LogDetailsPreservesSpecialChars(t *testing.T) {
	h := NewTestHarness(t)
	specialMsg := "Test with quotes \" ' and newline \n and \t tab"
	_ = h.LogEvent(BotEventLog{EventType: "auto_reply", SenderJID: "u1", IncomingMessage: specialMsg, Status: "success"})

	var retrieved string
	err := h.DB.QueryRow(`SELECT incoming_message FROM bot_event_logs WHERE event_type = 'auto_reply'`).Scan(&retrieved)
	require.NoError(t, err)
	assert.Equal(t, specialMsg, retrieved)
}

func TestTier2_Boundary_Feature14_MissingStatic404Graceful(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodGet, "/statics/non_existent.png", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTier2_Boundary_Feature14_PathTraversalAttemptRejected(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodGet, "/../../etc/passwd", nil)
	require.NoError(t, err)
	assert.Contains(t, []int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest}, resp.StatusCode)
}

func TestTier2_Boundary_Feature14_LargeStaticPayloadStream(t *testing.T) {
	h := NewTestHarness(t)
	resp, body, err := h.DoRequest(http.MethodGet, "/statics/large.dat", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, body)
}

func TestTier2_Boundary_Feature14_CacheControlHeaders(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTier2_Boundary_Feature14_OptionsCORSPreflight(t *testing.T) {
	h := NewTestHarness(t)
	resp, _, err := h.DoRequest(http.MethodOptions, "/bot/rules", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}
