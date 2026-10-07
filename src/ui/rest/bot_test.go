package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	domainBot "github.com/dresar/gowanew/domains/bot"
	domainChatStorage "github.com/dresar/gowanew/domains/chatstorage"
	botInfrastructure "github.com/dresar/gowanew/infrastructure/bot"
	"github.com/dresar/gowanew/pkg/utils"
	"github.com/dresar/gowanew/usecase"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type mockBotUsecase struct {
	rules      map[int64]*domainBot.Rule
	groupRules map[string]*domainBot.GroupRule
	aiConfig   *domainBot.AIConfig
	logs       []*domainBot.EventLog
	nextRuleID int64
	nextLogID  int64
}

func newMockBotUsecase() *mockBotUsecase {
	return &mockBotUsecase{
		rules:      make(map[int64]*domainBot.Rule),
		groupRules: make(map[string]*domainBot.GroupRule),
		aiConfig: &domainBot.AIConfig{
			ID:               1,
			Provider:         "openai",
			BaseURL:          "https://api.openai.com/v1",
			APIKey:           "test-key",
			Model:            "gpt-4o-mini",
			SystemPrompt:     "You are a test bot",
			Temperature:      0.7,
			TriggerPrefix:    "!ai",
			AutoReplyEnabled: false,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		logs:       make([]*domainBot.EventLog, 0),
		nextRuleID: 1,
		nextLogID:  1,
	}
}

func (m *mockBotUsecase) CreateRule(_ context.Context, req domainBot.CreateRuleRequest) (*domainBot.Rule, error) {
	if req.TriggerValue == "" || req.ResponseContent == "" {
		return nil, domainBot.ErrMissingTriggerValue
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	r := &domainBot.Rule{
		ID:              m.nextRuleID,
		TriggerType:     req.TriggerType,
		TriggerValue:    req.TriggerValue,
		Scope:           req.Scope,
		ResponseType:    req.ResponseType,
		ResponseContent: req.ResponseContent,
		MediaURL:        req.MediaURL,
		IsActive:        active,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	m.rules[m.nextRuleID] = r
	m.nextRuleID++
	return r, nil
}

func (m *mockBotUsecase) GetRuleByID(_ context.Context, id int64) (*domainBot.Rule, error) {
	r, ok := m.rules[id]
	if !ok {
		return nil, domainBot.ErrRuleNotFound
	}
	return r, nil
}

func (m *mockBotUsecase) ListRules(_ context.Context, filter domainBot.RuleFilter) ([]*domainBot.Rule, error) {
	res := make([]*domainBot.Rule, 0)
	for _, r := range m.rules {
		if filter.IsActive != nil && r.IsActive != *filter.IsActive {
			continue
		}
		if filter.Scope != nil && r.Scope != *filter.Scope {
			continue
		}
		res = append(res, r)
	}
	return res, nil
}

func (m *mockBotUsecase) UpdateRule(_ context.Context, id int64, req domainBot.UpdateRuleRequest) (*domainBot.Rule, error) {
	r, ok := m.rules[id]
	if !ok {
		return nil, domainBot.ErrRuleNotFound
	}
	if req.TriggerValue != nil {
		r.TriggerValue = *req.TriggerValue
	}
	if req.ResponseContent != nil {
		r.ResponseContent = *req.ResponseContent
	}
	if req.IsActive != nil {
		r.IsActive = *req.IsActive
	}
	r.UpdatedAt = time.Now()
	return r, nil
}

func (m *mockBotUsecase) DeleteRule(_ context.Context, id int64) error {
	if _, ok := m.rules[id]; !ok {
		return domainBot.ErrRuleNotFound
	}
	delete(m.rules, id)
	return nil
}

func (m *mockBotUsecase) DeleteRulesBulk(_ context.Context, ids []int64) (int64, error) {
	var count int64
	for _, id := range ids {
		if _, ok := m.rules[id]; ok {
			delete(m.rules, id)
			count++
		}
	}
	return count, nil
}

func (m *mockBotUsecase) ClearAllRules(_ context.Context) error {
	m.rules = make(map[int64]*domainBot.Rule)
	return nil
}

func (m *mockBotUsecase) ToggleRuleActive(_ context.Context, id int64) (*domainBot.Rule, error) {
	r, ok := m.rules[id]
	if !ok {
		return nil, domainBot.ErrRuleNotFound
	}
	r.IsActive = !r.IsActive
	return r, nil
}

func (m *mockBotUsecase) GetGroupRule(_ context.Context, groupJID string) (*domainBot.GroupRule, error) {
	gr, ok := m.groupRules[groupJID]
	if !ok {
		return nil, domainBot.ErrGroupRuleNotFound
	}
	return gr, nil
}

func (m *mockBotUsecase) ListGroupRules(_ context.Context) ([]*domainBot.GroupRule, error) {
	res := make([]*domainBot.GroupRule, 0)
	for _, gr := range m.groupRules {
		res = append(res, gr)
	}
	return res, nil
}

func (m *mockBotUsecase) UpsertGroupRule(_ context.Context, req domainBot.UpsertGroupRuleRequest) (*domainBot.GroupRule, error) {
	antiLink := false
	if req.AntiLinkEnabled != nil {
		antiLink = *req.AntiLinkEnabled
	}
	gr := &domainBot.GroupRule{
		GroupJID:        req.GroupJID,
		AntiLinkEnabled: antiLink,
	}
	m.groupRules[req.GroupJID] = gr
	return gr, nil
}

func (m *mockBotUsecase) DeleteGroupRule(_ context.Context, groupJID string) error {
	if _, ok := m.groupRules[groupJID]; !ok {
		return domainBot.ErrGroupRuleNotFound
	}
	delete(m.groupRules, groupJID)
	return nil
}

func (m *mockBotUsecase) GetAIConfig(_ context.Context) (*domainBot.AIConfig, error) {
	return m.aiConfig, nil
}

func (m *mockBotUsecase) UpdateAIConfig(_ context.Context, req domainBot.UpdateAIConfigRequest) (*domainBot.AIConfig, error) {
	if req.Model != nil {
		m.aiConfig.Model = *req.Model
	}
	if req.AutoReplyEnabled != nil {
		m.aiConfig.AutoReplyEnabled = *req.AutoReplyEnabled
	}
	return m.aiConfig, nil
}

func (m *mockBotUsecase) ListAIPersonas(_ context.Context) ([]*domainBot.AIPersona, error) {
	return []*domainBot.AIPersona{
		{
			ID:               1,
			PhoneNumber:      "6285216149732",
			ContactName:      "Indah 🧕🌿💝",
			Relationship:     "pacar",
			CustomPrompt:     "Halo sayang",
			AutoReplyEnabled: true,
			UseMemory:        true,
			IsActive:         true,
		},
	}, nil
}

func (m *mockBotUsecase) GetAIPersonaByID(_ context.Context, id int64) (*domainBot.AIPersona, error) {
	if id == 1 {
		return &domainBot.AIPersona{
			ID:               1,
			PhoneNumber:      "6285216149732",
			ContactName:      "Indah 🧕🌿💝",
			Relationship:     "pacar",
			CustomPrompt:     "Halo sayang",
			AutoReplyEnabled: true,
			UseMemory:        true,
			IsActive:         true,
		}, nil
	}
	return nil, domainBot.ErrAIPersonaNotFound
}

func (m *mockBotUsecase) GetAIPersonaByPhone(_ context.Context, phone string) (*domainBot.AIPersona, error) {
	if phone == "6285216149732" {
		return &domainBot.AIPersona{
			ID:               1,
			PhoneNumber:      "6285216149732",
			ContactName:      "Indah 🧕🌿💝",
			Relationship:     "pacar",
			CustomPrompt:     "Halo sayang",
			AutoReplyEnabled: true,
			UseMemory:        true,
			IsActive:         true,
		}, nil
	}
	return nil, domainBot.ErrAIPersonaNotFound
}

func (m *mockBotUsecase) CreateAIPersona(_ context.Context, req domainBot.CreateAIPersonaRequest) (*domainBot.AIPersona, error) {
	return &domainBot.AIPersona{
		ID:               2,
		PhoneNumber:      req.PhoneNumber,
		ContactName:      req.ContactName,
		Relationship:     req.Relationship,
		CustomPrompt:     req.CustomPrompt,
		AutoReplyEnabled: true,
		UseMemory:        true,
		IsActive:         true,
	}, nil
}

func (m *mockBotUsecase) UpdateAIPersona(_ context.Context, id int64, req domainBot.UpdateAIPersonaRequest) (*domainBot.AIPersona, error) {
	if id != 1 {
		return nil, domainBot.ErrAIPersonaNotFound
	}
	return &domainBot.AIPersona{
		ID:               1,
		PhoneNumber:      "6285216149732",
		ContactName:      "Indah",
		Relationship:     "pacar",
		CustomPrompt:     "Halo sayang",
		AutoReplyEnabled: true,
		UseMemory:        true,
		IsActive:         true,
	}, nil
}

func (m *mockBotUsecase) DeleteAIPersona(_ context.Context, id int64) error {
	if id != 1 {
		return domainBot.ErrAIPersonaNotFound
	}
	return nil
}

func (m *mockBotUsecase) ChatWithAI(_ context.Context, req domainBot.ChatRequest) (*domainBot.ChatResult, error) {
	return &domainBot.ChatResult{
		Reply:     "Echo: " + req.Message,
		Model:     "gpt-4o-mini",
		LatencyMS: 50,
		Usage: domainBot.ChatUsage{
			PromptTokens:     10,
			CompletionTokens: 10,
			TotalTokens:      20,
		},
	}, nil
}

func (m *mockBotUsecase) GetTools(_ context.Context) ([]domainBot.ToolDefinition, error) {
	return []domainBot.ToolDefinition{
		{Name: "send_message", Description: "Send text"},
		{Name: "manage_group", Description: "Manage group"},
		{Name: "query_chats", Description: "Query chats"},
		{Name: "update_rules", Description: "Update rules"},
	}, nil
}

func (m *mockBotUsecase) ExecuteTool(_ context.Context, _ *whatsmeow.Client, _ domainChatStorage.IChatStorageRepository, req domainBot.ToolRequest) (*domainBot.ToolResult, error) {
	return &domainBot.ToolResult{
		Tool:      req.Tool,
		Status:    "success",
		LatencyMS: 10,
		Output:    map[string]any{"executed": true},
	}, nil
}

func (m *mockBotUsecase) CreateEventLog(_ context.Context, dto domainBot.CreateEventLogDTO) (*domainBot.EventLog, error) {
	log := &domainBot.EventLog{
		ID:        m.nextLogID,
		EventType: dto.EventType,
		SenderJID: dto.SenderJID,
		Status:    dto.Status,
	}
	m.logs = append(m.logs, log)
	m.nextLogID++
	return log, nil
}

func (m *mockBotUsecase) ListEventLogs(_ context.Context, _ domainBot.EventLogFilter) ([]*domainBot.EventLog, int64, error) {
	return m.logs, int64(len(m.logs)), nil
}

func (m *mockBotUsecase) ClearEventLogs(_ context.Context) error {
	m.logs = make([]*domainBot.EventLog, 0)
	return nil
}

func (m *mockBotUsecase) MatchRule(_ []*domainBot.Rule, _ string, _ bool) *domainBot.Rule {
	return nil
}

func (m *mockBotUsecase) MatchRuleForSender(_ []*domainBot.Rule, _ string, _ bool, _ string) *domainBot.Rule {
	return nil
}

func (m *mockBotUsecase) AutoTagPacarRules(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockBotUsecase) ModerateMessage(_ context.Context, _, _ string, _ string) (*usecase.ModerationResult, bool) {
	return nil, false
}

func (m *mockBotUsecase) FormatWelcome(_, _, _ string) string {
	return ""
}

func (m *mockBotUsecase) FormatFarewell(_, _, _ string) string {
	return ""
}

func (m *mockBotUsecase) IsBotAdmin(_ context.Context, _ *whatsmeow.Client, _ types.JID) bool {
	return false
}

func (m *mockBotUsecase) ExtractIncomingText(_ *events.Message) string {
	return ""
}

func (m *mockBotUsecase) ShouldIgnoreMessage(_ *events.Message, _ *whatsmeow.Client) bool {
	return false
}

func (m *mockBotUsecase) DispatchResponse(_ context.Context, _ *whatsmeow.Client, _ types.JID, _ *domainBot.Rule) (string, error) {
	return "", nil
}

func (m *mockBotUsecase) HandleMessage(_ context.Context, _ *events.Message, _ *whatsmeow.Client, _ domainChatStorage.IChatStorageRepository) (bool, error) {
	return false, nil
}

func (m *mockBotUsecase) HandleGroupInfo(_ context.Context, _ *events.GroupInfo, _ *whatsmeow.Client) {}

func setupTestBotApp() (*fiber.App, *mockBotUsecase) {
	app := fiber.New()
	mock := newMockBotUsecase()
	InitRestBot(app, mock, nil, nil)
	return app, mock
}

func doReq(app *fiber.App, method, url string, body any) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, url, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		return nil, nil, err
	}
	out, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, out, err
}

func TestBotRulesCRUD(t *testing.T) {
	app, _ := setupTestBotApp()

	createPayload := domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "hello",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "world",
	}
	resp, body, err := doReq(app, http.MethodPost, "/bot/rules", createPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var createData utils.ResponseData
	require.NoError(t, json.Unmarshal(body, &createData))
	assert.Equal(t, "SUCCESS", createData.Code)

	resp, body, err = doReq(app, http.MethodGet, "/bot/rules", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body, err = doReq(app, http.MethodGet, "/bot/rules/1", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	newVal := "updated"
	updatePayload := domainBot.UpdateRuleRequest{ResponseContent: &newVal}
	resp, body, err = doReq(app, http.MethodPut, "/bot/rules/1", updatePayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodPatch, "/bot/rules/1/toggle", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body, err = doReq(app, http.MethodDelete, "/bot/rules/1", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body, err = doReq(app, http.MethodGet, "/bot/rules/1", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestBotGroupRulesCRUD(t *testing.T) {
	app, _ := setupTestBotApp()

	groupJID := "test_grp@g.us"
	antiLink := true
	upsertPayload := domainBot.UpsertGroupRuleRequest{
		GroupJID:        groupJID,
		AntiLinkEnabled: &antiLink,
	}
	resp, _, err := doReq(app, http.MethodPost, "/bot/group-rules", upsertPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodGet, "/bot/group-rules", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodGet, "/bot/group-rules/"+groupJID, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodDelete, "/bot/group-rules/"+groupJID, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodGet, "/bot/group-rules/"+groupJID, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestBotAIAssistantAndTools(t *testing.T) {
	app, _ := setupTestBotApp()

	resp, _, err := doReq(app, http.MethodGet, "/bot/ai/config", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	model := "gpt-4o"
	resp, _, err = doReq(app, http.MethodPut, "/bot/ai/config", domainBot.UpdateAIConfigRequest{Model: &model})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	chatPayload := domainBot.ChatRequest{Message: "hello ai"}
	resp, _, err = doReq(app, http.MethodPost, "/bot/ai/chat", chatPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodGet, "/bot/ai/tools", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	toolPayload := domainBot.ToolRequest{Tool: "send_message", Parameters: map[string]any{"recipient": "123", "message": "hi"}}
	resp, _, err = doReq(app, http.MethodPost, "/bot/ai/tools", toolPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodPost, "/bot/ai/execute", toolPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBotLogsEndpoints(t *testing.T) {
	app, mock := setupTestBotApp()

	_, _ = mock.CreateEventLog(context.Background(), domainBot.CreateEventLogDTO{
		EventType: domainBot.EventTypeAutoReply,
		SenderJID: "user1",
		Status:    domainBot.LogStatusSuccess,
	})

	resp, _, err := doReq(app, http.MethodGet, "/bot/logs", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodDelete, "/bot/logs", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodGet, "/bot/logs", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBotRealSQLiteIntegration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "bot.db")
	db, err := botInfrastructure.OpenBotDB(dbPath, 1)
	require.NoError(t, err)
	defer db.Close()

	repo := botInfrastructure.NewSQLiteRepository(db)
	require.NoError(t, repo.InitializeSchema(context.Background()))
	service := usecase.NewBotService(repo)

	app := fiber.New()
	InitRestBot(app, service, nil, nil)

	createPayload := domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "real_db_test",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "response from real db",
	}
	resp, _, err := doReq(app, http.MethodPost, "/bot/rules", createPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, body, err := doReq(app, http.MethodGet, "/bot/rules?search=real_db_test", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var listData utils.ResponseData
	require.NoError(t, json.Unmarshal(body, &listData))
	rulesList, ok := listData.Results.([]any)
	require.True(t, ok)
	assert.Len(t, rulesList, 1)
}

func TestBotAIPersonasEndpoints(t *testing.T) {
	app, _ := setupTestBotApp()

	resp, body, err := doReq(app, http.MethodGet, "/bot/ai/personas", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var listData utils.ResponseData
	require.NoError(t, json.Unmarshal(body, &listData))
	assert.Equal(t, "SUCCESS", listData.Code)

	resp, _, err = doReq(app, http.MethodGet, "/bot/ai/personas/1", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	newPersona := domainBot.CreateAIPersonaRequest{
		PhoneNumber:  "628123456789",
		ContactName:  "Budi",
		Relationship: "teman",
		CustomPrompt: "Balas santai",
	}
	resp, _, err = doReq(app, http.MethodPost, "/bot/ai/personas", newPersona)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	updatePrompt := "Update prompt"
	updatePersona := domainBot.UpdateAIPersonaRequest{
		CustomPrompt: &updatePrompt,
	}
	resp, _, err = doReq(app, http.MethodPut, "/bot/ai/personas/1", updatePersona)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodDelete, "/bot/ai/personas/1", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBotHandler_BulkDeleteRules(t *testing.T) {
	app, mock := setupTestBotApp()

	_, _ = mock.CreateRule(context.Background(), domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "test1",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "resp1",
	})
	_, _ = mock.CreateRule(context.Background(), domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "test2",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "resp2",
	})

	resp, _, err := doReq(app, http.MethodDelete, "/bot/rules", BulkDeleteRulesPayload{IDs: []int64{1}})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _, err = doReq(app, http.MethodDelete, "/bot/rules?all=true", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

