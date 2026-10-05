package e2e

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type BotRule struct {
	ID              int64     `json:"id"`
	TriggerType     string    `json:"trigger_type"`
	TriggerValue    string    `json:"trigger_value"`
	Scope           string    `json:"scope"`
	ResponseType    string    `json:"response_type"`
	ResponseContent string    `json:"response_content"`
	MediaURL        string    `json:"media_url"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type BotGroupRule struct {
	ID               int64     `json:"id"`
	GroupJID         string    `json:"group_jid"`
	AntiLinkEnabled  bool      `json:"anti_link_enabled"`
	WelcomeEnabled   bool      `json:"welcome_enabled"`
	WelcomeTemplate  string    `json:"welcome_template"`
	FarewellEnabled  bool      `json:"farewell_enabled"`
	FarewellTemplate string    `json:"farewell_template"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type BotAIConfig struct {
	ID               int64     `json:"id"`
	Provider         string    `json:"provider"`
	BaseURL          string    `json:"base_url"`
	APIKey           string    `json:"api_key"`
	Model            string    `json:"model"`
	SystemPrompt     string    `json:"system_prompt"`
	Temperature      float64   `json:"temperature"`
	TriggerPrefix    string    `json:"trigger_prefix"`
	AutoReplyEnabled bool      `json:"auto_reply_enabled"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type BotEventLog struct {
	ID              int64     `json:"id"`
	EventType       string    `json:"event_type"`
	RuleID          *int64    `json:"rule_id,omitempty"`
	SenderJID       string    `json:"sender_jid"`
	GroupJID        string    `json:"group_jid"`
	IncomingMessage string    `json:"incoming_message"`
	ResponseMessage string    `json:"response_message"`
	LatencyMS       int64     `json:"latency_ms"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type APIResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Results any    `json:"results,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message     string        `json:"message"`
	ChatHistory []ChatMessage `json:"chat_history,omitempty"`
	Model       string        `json:"model,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type ChatResult struct {
	Reply     string `json:"reply"`
	Model     string `json:"model"`
	LatencyMS int64  `json:"latency_ms"`
	Usage     struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type ToolRequest struct {
	Tool       string         `json:"tool"`
	Parameters map[string]any `json:"parameters"`
}

type ToolResult struct {
	Tool      string `json:"tool"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Output    any    `json:"output"`
}

type MockMessageSent struct {
	Recipient string
	Message   string
	MediaType string
	MediaURL  string
}

type MockMessageRevoked struct {
	GroupJID  string
	SenderJID string
	MessageID string
}

type MockParticipantAction struct {
	GroupJID     string
	Action       string
	Participants []string
}

type MockWhatsAppDispatcher struct {
	mu           sync.Mutex
	SentMessages []MockMessageSent
	Revocations  []MockMessageRevoked
	GroupActions []MockParticipantAction
}

func NewMockWhatsAppDispatcher() *MockWhatsAppDispatcher {
	return &MockWhatsAppDispatcher{
		SentMessages: make([]MockMessageSent, 0),
		Revocations:  make([]MockMessageRevoked, 0),
		GroupActions: make([]MockParticipantAction, 0),
	}
}

func (m *MockWhatsAppDispatcher) SendText(recipient string, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentMessages = append(m.SentMessages, MockMessageSent{
		Recipient: recipient,
		Message:   message,
		MediaType: "text",
	})
}

func (m *MockWhatsAppDispatcher) SendMedia(recipient string, caption string, mediaType string, mediaURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentMessages = append(m.SentMessages, MockMessageSent{
		Recipient: recipient,
		Message:   caption,
		MediaType: mediaType,
		MediaURL:  mediaURL,
	})
}

func (m *MockWhatsAppDispatcher) RevokeMessage(groupJID string, senderJID string, messageID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Revocations = append(m.Revocations, MockMessageRevoked{
		GroupJID:  groupJID,
		SenderJID: senderJID,
		MessageID: messageID,
	})
}

func (m *MockWhatsAppDispatcher) ManageParticipant(groupJID string, action string, participants []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GroupActions = append(m.GroupActions, MockParticipantAction{
		GroupJID:     groupJID,
		Action:       action,
		Participants: participants,
	})
}

type OpenAIMockResponse struct {
	MatchPrompt string
	ReplyText   string
	ToolCalls   []any
	StatusCode  int
}

type TestHarness struct {
	T               *testing.T
	DB              *sql.DB
	DBPath          string
	OpenAIMock      *httptest.Server
	OpenAIResponses []OpenAIMockResponse
	OpenAIMu        sync.Mutex
	Dispatcher      *MockWhatsAppDispatcher
	Server          *httptest.Server
	BaseURL         string
	HTTPClient      *http.Client
}

func NewTestHarness(t *testing.T) *TestHarness {
	h := &TestHarness{
		T:          t,
		Dispatcher: NewMockWhatsAppDispatcher(),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	h.setupMockOpenAI()
	h.setupDatabase()
	h.setupServer()

	t.Cleanup(func() {
		h.Close()
	})

	return h
}

func (h *TestHarness) Close() {
	if h.Server != nil {
		h.Server.Close()
	}
	if h.OpenAIMock != nil {
		h.OpenAIMock.Close()
	}
	if h.DB != nil {
		_ = h.DB.Close()
	}
}

func (h *TestHarness) setupMockOpenAI() {
	h.OpenAIMock = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.OpenAIMu.Lock()
		defer h.OpenAIMu.Unlock()

		authHeader := r.Header.Get("Authorization")
		if authHeader == "Bearer invalid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": {"message": "Incorrect API key provided"}}`))
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		bodyStr := string(bodyBytes)
		for _, mockResp := range h.OpenAIResponses {
			if mockResp.MatchPrompt != "" && strings.Contains(bodyStr, mockResp.MatchPrompt) {
				if mockResp.StatusCode != 0 {
					w.WriteHeader(mockResp.StatusCode)
				}
				respPayload := map[string]any{
					"id":      "chatcmpl-mock",
					"object":  "chat.completion",
					"created": time.Now().Unix(),
					"model":   "gpt-4o-mini",
					"choices": []map[string]any{
						{
							"index": 0,
							"message": map[string]any{
								"role":       "assistant",
								"content":    mockResp.ReplyText,
								"tool_calls": mockResp.ToolCalls,
							},
							"finish_reason": "stop",
						},
					},
					"usage": map[string]any{
						"prompt_tokens":     15,
						"completion_tokens": 10,
						"total_tokens":      25,
					},
				}
				_ = json.NewEncoder(w).Encode(respPayload)
				return
			}
		}

		respPayload := map[string]any{
			"id":      "chatcmpl-mock-default",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "gpt-4o-mini",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Autonomous AI assistant response.",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     12,
				"completion_tokens": 8,
				"total_tokens":      20,
			},
		}
		_ = json.NewEncoder(w).Encode(respPayload)
	}))
}

func (h *TestHarness) SetOpenAIResponse(prompt string, reply string, toolCalls []any, status int) {
	h.OpenAIMu.Lock()
	defer h.OpenAIMu.Unlock()
	h.OpenAIResponses = append([]OpenAIMockResponse{{
		MatchPrompt: prompt,
		ReplyText:   reply,
		ToolCalls:   toolCalls,
		StatusCode:  status,
	}}, h.OpenAIResponses...)
}

func (h *TestHarness) setupDatabase() {
	tempDir := h.T.TempDir()
	h.DBPath = filepath.Join(tempDir, "bot.db")

	uri := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(30000)&_pragma=synchronous(1)&_pragma=foreign_keys(1)", h.DBPath)
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		h.T.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	h.DB = db

	h.ApplyMigrations()
}

func (h *TestHarness) ApplyMigrations() {
	migrationSQL := `
	CREATE TABLE IF NOT EXISTS bot_schema_info (
		version INTEGER PRIMARY KEY,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS bot_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trigger_type TEXT NOT NULL CHECK (trigger_type IN ('exact', 'contains', 'starts_with', 'regex')),
		trigger_value TEXT NOT NULL,
		scope TEXT NOT NULL DEFAULT 'all' CHECK (scope IN ('private', 'group', 'all')),
		response_type TEXT NOT NULL DEFAULT 'text' CHECK (response_type IN ('text', 'media')),
		response_content TEXT NOT NULL,
		media_url TEXT NOT NULL DEFAULT '',
		is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_bot_rules_active ON bot_rules(is_active);
	CREATE INDEX IF NOT EXISTS idx_bot_rules_trigger_type ON bot_rules(trigger_type);
	CREATE INDEX IF NOT EXISTS idx_bot_rules_scope ON bot_rules(scope);

	CREATE TABLE IF NOT EXISTS bot_group_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_jid TEXT NOT NULL UNIQUE,
		anti_link_enabled INTEGER NOT NULL DEFAULT 0 CHECK (anti_link_enabled IN (0, 1)),
		welcome_enabled INTEGER NOT NULL DEFAULT 0 CHECK (welcome_enabled IN (0, 1)),
		welcome_template TEXT NOT NULL DEFAULT '',
		farewell_enabled INTEGER NOT NULL DEFAULT 0 CHECK (farewell_enabled IN (0, 1)),
		farewell_template TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_bot_group_rules_jid ON bot_group_rules(group_jid);

	CREATE TABLE IF NOT EXISTS bot_ai_config (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		provider TEXT NOT NULL DEFAULT 'openai',
		base_url TEXT NOT NULL DEFAULT 'https://api.openai.com/v1',
		api_key TEXT NOT NULL DEFAULT '',
		model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
		system_prompt TEXT NOT NULL DEFAULT '',
		temperature REAL NOT NULL DEFAULT 0.7,
		trigger_prefix TEXT NOT NULL DEFAULT '!ai',
		auto_reply_enabled INTEGER NOT NULL DEFAULT 0 CHECK (auto_reply_enabled IN (0, 1)),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS bot_event_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL CHECK (event_type IN ('auto_reply', 'group_moderation', 'ai_chat', 'ai_tool', 'error')),
		rule_id INTEGER,
		sender_jid TEXT NOT NULL,
		group_jid TEXT NOT NULL DEFAULT '',
		incoming_message TEXT NOT NULL DEFAULT '',
		response_message TEXT NOT NULL DEFAULT '',
		latency_ms INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'ignored', 'rate_limited')),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_bot_event_logs_event_type ON bot_event_logs(event_type);
	CREATE INDEX IF NOT EXISTS idx_bot_event_logs_created_at ON bot_event_logs(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_bot_event_logs_sender ON bot_event_logs(sender_jid);
	CREATE INDEX IF NOT EXISTS idx_bot_event_logs_group ON bot_event_logs(group_jid);
	CREATE INDEX IF NOT EXISTS idx_bot_event_logs_status ON bot_event_logs(status);

	INSERT OR IGNORE INTO bot_schema_info (version, updated_at) VALUES (1, CURRENT_TIMESTAMP);

	INSERT OR IGNORE INTO bot_ai_config (
		id, provider, base_url, api_key, model, system_prompt, temperature, trigger_prefix, auto_reply_enabled, created_at, updated_at
	) VALUES (
		1, 'openai', '` + h.OpenAIMock.URL + `', 'mock-key', 'gpt-4o-mini',
		'You are a helpful WhatsApp assistant.', 0.7, '!ai', 0,
		CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
	);
	`
	_, err := h.DB.Exec(migrationSQL)
	if err != nil {
		h.T.Fatalf("apply migrations failed: %v", err)
	}
}

func (h *TestHarness) setupServer() {
	envTarget := os.Getenv("BOT_E2E_BASE_URL")
	if envTarget != "" {
		h.BaseURL = envTarget
		return
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/bot/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			activeParam := r.URL.Query().Get("active")
			scopeParam := r.URL.Query().Get("scope")
			searchParam := r.URL.Query().Get("search")

			query := `SELECT id, trigger_type, trigger_value, scope, response_type, response_content, media_url, is_active, created_at, updated_at FROM bot_rules WHERE 1=1`
			var args []any
			if activeParam != "" {
				isActive := activeParam == "true" || activeParam == "1"
				query += ` AND is_active = ?`
				if isActive {
					args = append(args, 1)
				} else {
					args = append(args, 0)
				}
			}
			if scopeParam != "" {
				query += ` AND scope = ?`
				args = append(args, scopeParam)
			}
			if searchParam != "" {
				query += ` AND (trigger_value LIKE ? OR response_content LIKE ?)`
				args = append(args, "%"+searchParam+"%", "%"+searchParam+"%")
			}
			query += ` ORDER BY id ASC`

			rows, err := h.DB.Query(query, args...)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "ERROR", Message: err.Error()})
				return
			}
			defer rows.Close()

			rules := make([]BotRule, 0)
			for rows.Next() {
				var rule BotRule
				var isActiveInt int
				var createdAtStr, updatedAtStr string
				if err := rows.Scan(&rule.ID, &rule.TriggerType, &rule.TriggerValue, &rule.Scope, &rule.ResponseType, &rule.ResponseContent, &rule.MediaURL, &isActiveInt, &createdAtStr, &updatedAtStr); err == nil {
					rule.IsActive = isActiveInt == 1
					rule.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
					rule.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
					rules = append(rules, rule)
				}
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "rules fetched", Results: rules})
			return
		}

		if r.Method == http.MethodPost {
			var body struct {
				TriggerType     string `json:"trigger_type"`
				TriggerValue    string `json:"trigger_value"`
				Scope           string `json:"scope"`
				ResponseType    string `json:"response_type"`
				ResponseContent string `json:"response_content"`
				MediaURL        string `json:"media_url"`
				IsActive        *bool  `json:"is_active"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid json body"})
				return
			}
			if body.TriggerValue == "" || body.ResponseContent == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "trigger_value and response_content are required"})
				return
			}
			if body.Scope == "" {
				body.Scope = "all"
			}
			if body.ResponseType == "" {
				body.ResponseType = "text"
			}
			isActiveInt := 1
			if body.IsActive != nil && !*body.IsActive {
				isActiveInt = 0
			}

			res, err := h.DB.Exec(
				`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, media_url, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
				body.TriggerType, body.TriggerValue, body.Scope, body.ResponseType, body.ResponseContent, body.MediaURL, isActiveInt,
			)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "ERROR", Message: err.Error()})
				return
			}
			id, _ := res.LastInsertId()
			createdRule := BotRule{
				ID:              id,
				TriggerType:     body.TriggerType,
				TriggerValue:    body.TriggerValue,
				Scope:           body.Scope,
				ResponseType:    body.ResponseType,
				ResponseContent: body.ResponseContent,
				MediaURL:        body.MediaURL,
				IsActive:        isActiveInt == 1,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "rule created", Results: createdRule})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/bot/rules/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/bot/rules/")
		if strings.HasSuffix(path, "/toggle") && r.Method == http.MethodPatch {
			idStr := strings.TrimSuffix(path, "/toggle")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid rule id"})
				return
			}
			res, err := h.DB.Exec(`UPDATE bot_rules SET is_active = CASE WHEN is_active = 1 THEN 0 ELSE 1 END, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "NOT_FOUND", Message: "rule not found"})
				return
			}
			var rule BotRule
			var isActiveInt int
			var createdAtStr, updatedAtStr string
			_ = h.DB.QueryRow(`SELECT id, trigger_type, trigger_value, scope, response_type, response_content, media_url, is_active, created_at, updated_at FROM bot_rules WHERE id = ?`, id).
				Scan(&rule.ID, &rule.TriggerType, &rule.TriggerValue, &rule.Scope, &rule.ResponseType, &rule.ResponseContent, &rule.MediaURL, &isActiveInt, &createdAtStr, &updatedAtStr)
			rule.IsActive = isActiveInt == 1
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "rule toggled", Results: rule})
			return
		}

		id, err := strconv.ParseInt(path, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid rule id"})
			return
		}

		if r.Method == http.MethodGet {
			var rule BotRule
			var isActiveInt int
			var createdAtStr, updatedAtStr string
			err := h.DB.QueryRow(`SELECT id, trigger_type, trigger_value, scope, response_type, response_content, media_url, is_active, created_at, updated_at FROM bot_rules WHERE id = ?`, id).
				Scan(&rule.ID, &rule.TriggerType, &rule.TriggerValue, &rule.Scope, &rule.ResponseType, &rule.ResponseContent, &rule.MediaURL, &isActiveInt, &createdAtStr, &updatedAtStr)
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "NOT_FOUND", Message: "rule not found"})
				return
			}
			rule.IsActive = isActiveInt == 1
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "rule retrieved", Results: rule})
			return
		}

		if r.Method == http.MethodPut {
			var body struct {
				TriggerType     string `json:"trigger_type"`
				TriggerValue    string `json:"trigger_value"`
				Scope           string `json:"scope"`
				ResponseType    string `json:"response_type"`
				ResponseContent string `json:"response_content"`
				MediaURL        string `json:"media_url"`
				IsActive        *bool  `json:"is_active"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid json"})
				return
			}
			isActiveInt := 1
			if body.IsActive != nil && !*body.IsActive {
				isActiveInt = 0
			}
			res, err := h.DB.Exec(
				`UPDATE bot_rules SET trigger_type = ?, trigger_value = ?, scope = ?, response_type = ?, response_content = ?, media_url = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
				body.TriggerType, body.TriggerValue, body.Scope, body.ResponseType, body.ResponseContent, body.MediaURL, isActiveInt, id,
			)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "ERROR", Message: err.Error()})
				return
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "NOT_FOUND", Message: "rule not found"})
				return
			}
			updatedRule := BotRule{
				ID:              id,
				TriggerType:     body.TriggerType,
				TriggerValue:    body.TriggerValue,
				Scope:           body.Scope,
				ResponseType:    body.ResponseType,
				ResponseContent: body.ResponseContent,
				MediaURL:        body.MediaURL,
				IsActive:        isActiveInt == 1,
				UpdatedAt:       time.Now(),
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "rule updated", Results: updatedRule})
			return
		}

		if r.Method == http.MethodDelete {
			res, err := h.DB.Exec(`DELETE FROM bot_rules WHERE id = ?`, id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "NOT_FOUND", Message: "rule not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "rule deleted", Results: map[string]any{"id": id, "deleted": true}})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/bot/group-rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodGet {
			rows, err := h.DB.Query(`SELECT id, group_jid, anti_link_enabled, welcome_enabled, welcome_template, farewell_enabled, farewell_template, created_at, updated_at FROM bot_group_rules ORDER BY id ASC`)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			groupRules := make([]BotGroupRule, 0)
			for rows.Next() {
				var gr BotGroupRule
				var antiLink, welcome, farewell int
				var cStr, uStr string
				if err := rows.Scan(&gr.ID, &gr.GroupJID, &antiLink, &welcome, &gr.WelcomeTemplate, &farewell, &gr.FarewellTemplate, &cStr, &uStr); err == nil {
					gr.AntiLinkEnabled = antiLink == 1
					gr.WelcomeEnabled = welcome == 1
					gr.FarewellEnabled = farewell == 1
					groupRules = append(groupRules, gr)
				}
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "group rules fetched", Results: groupRules})
			return
		}

		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			var body struct {
				GroupJID         string `json:"group_jid"`
				AntiLinkEnabled  bool   `json:"anti_link_enabled"`
				WelcomeEnabled   bool   `json:"welcome_enabled"`
				WelcomeTemplate  string `json:"welcome_template"`
				FarewellEnabled  bool   `json:"farewell_enabled"`
				FarewellTemplate string `json:"farewell_template"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid json"})
				return
			}
			if body.GroupJID == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "group_jid is required"})
				return
			}

			antiLinkInt := 0
			if body.AntiLinkEnabled {
				antiLinkInt = 1
			}
			welcomeInt := 0
			if body.WelcomeEnabled {
				welcomeInt = 1
			}
			farewellInt := 0
			if body.FarewellEnabled {
				farewellInt = 1
			}

			_, err := h.DB.Exec(`
				INSERT INTO bot_group_rules (group_jid, anti_link_enabled, welcome_enabled, welcome_template, farewell_enabled, farewell_template, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				ON CONFLICT(group_jid) DO UPDATE SET
					anti_link_enabled = excluded.anti_link_enabled,
					welcome_enabled = excluded.welcome_enabled,
					welcome_template = excluded.welcome_template,
					farewell_enabled = excluded.farewell_enabled,
					farewell_template = excluded.farewell_template,
					updated_at = CURRENT_TIMESTAMP
			`, body.GroupJID, antiLinkInt, welcomeInt, body.WelcomeTemplate, farewellInt, body.FarewellTemplate)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "ERROR", Message: err.Error()})
				return
			}

			var gr BotGroupRule
			var aInt, wInt, fInt int
			var cStr, uStr string
			_ = h.DB.QueryRow(`SELECT id, group_jid, anti_link_enabled, welcome_enabled, welcome_template, farewell_enabled, farewell_template, created_at, updated_at FROM bot_group_rules WHERE group_jid = ?`, body.GroupJID).
				Scan(&gr.ID, &gr.GroupJID, &aInt, &wInt, &gr.WelcomeTemplate, &fInt, &gr.FarewellTemplate, &cStr, &uStr)
			gr.AntiLinkEnabled = aInt == 1
			gr.WelcomeEnabled = wInt == 1
			gr.FarewellEnabled = fInt == 1

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "group rule saved", Results: gr})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/bot/group-rules/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		groupJID := strings.TrimPrefix(r.URL.Path, "/bot/group-rules/")
		if groupJID == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodGet {
			var gr BotGroupRule
			var aInt, wInt, fInt int
			var cStr, uStr string
			err := h.DB.QueryRow(`SELECT id, group_jid, anti_link_enabled, welcome_enabled, welcome_template, farewell_enabled, farewell_template, created_at, updated_at FROM bot_group_rules WHERE group_jid = ?`, groupJID).
				Scan(&gr.ID, &gr.GroupJID, &aInt, &wInt, &gr.WelcomeTemplate, &fInt, &gr.FarewellTemplate, &cStr, &uStr)
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "NOT_FOUND", Message: "group rule not found"})
				return
			}
			gr.AntiLinkEnabled = aInt == 1
			gr.WelcomeEnabled = wInt == 1
			gr.FarewellEnabled = fInt == 1
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "group rule retrieved", Results: gr})
			return
		}

		if r.Method == http.MethodDelete {
			res, err := h.DB.Exec(`DELETE FROM bot_group_rules WHERE group_jid = ?`, groupJID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			affected, _ := res.RowsAffected()
			if affected == 0 {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "NOT_FOUND", Message: "group rule not found"})
				return
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "group rule deleted", Results: map[string]any{"group_jid": groupJID, "deleted": true}})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/bot/ai/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodGet {
			var cfg BotAIConfig
			var autoReplyInt int
			var cStr, uStr string
			err := h.DB.QueryRow(`SELECT id, provider, base_url, api_key, model, system_prompt, temperature, trigger_prefix, auto_reply_enabled, created_at, updated_at FROM bot_ai_config WHERE id = 1`).
				Scan(&cfg.ID, &cfg.Provider, &cfg.BaseURL, &cfg.APIKey, &cfg.Model, &cfg.SystemPrompt, &cfg.Temperature, &cfg.TriggerPrefix, &autoReplyInt, &cStr, &uStr)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cfg.AutoReplyEnabled = autoReplyInt == 1
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "ai config retrieved", Results: cfg})
			return
		}

		if r.Method == http.MethodPut {
			var body struct {
				Provider         string   `json:"provider"`
				BaseURL          string   `json:"base_url"`
				APIKey           string   `json:"api_key"`
				Model            string   `json:"model"`
				SystemPrompt     string   `json:"system_prompt"`
				Temperature      *float64 `json:"temperature"`
				TriggerPrefix    string   `json:"trigger_prefix"`
				AutoReplyEnabled *bool    `json:"auto_reply_enabled"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid json"})
				return
			}

			var current BotAIConfig
			var currentAutoReply int
			var cStr, uStr string
			_ = h.DB.QueryRow(`SELECT id, provider, base_url, api_key, model, system_prompt, temperature, trigger_prefix, auto_reply_enabled, created_at, updated_at FROM bot_ai_config WHERE id = 1`).
				Scan(&current.ID, &current.Provider, &current.BaseURL, &current.APIKey, &current.Model, &current.SystemPrompt, &current.Temperature, &current.TriggerPrefix, &currentAutoReply, &cStr, &uStr)

			if body.Provider != "" {
				current.Provider = body.Provider
			}
			if body.BaseURL != "" {
				current.BaseURL = body.BaseURL
			}
			if body.APIKey != "" {
				current.APIKey = body.APIKey
			}
			if body.Model != "" {
				current.Model = body.Model
			}
			if body.SystemPrompt != "" {
				current.SystemPrompt = body.SystemPrompt
			}
			if body.Temperature != nil {
				current.Temperature = *body.Temperature
			}
			if body.TriggerPrefix != "" {
				current.TriggerPrefix = body.TriggerPrefix
			}
			if body.AutoReplyEnabled != nil {
				current.AutoReplyEnabled = *body.AutoReplyEnabled
			}

			autoReplyInt := 0
			if current.AutoReplyEnabled {
				autoReplyInt = 1
			}

			_, err := h.DB.Exec(`
				UPDATE bot_ai_config SET
					provider = ?, base_url = ?, api_key = ?, model = ?,
					system_prompt = ?, temperature = ?, trigger_prefix = ?,
					auto_reply_enabled = ?, updated_at = CURRENT_TIMESTAMP
				WHERE id = 1
			`, current.Provider, current.BaseURL, current.APIKey, current.Model, current.SystemPrompt, current.Temperature, current.TriggerPrefix, autoReplyInt)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "ai config updated", Results: current})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/bot/ai/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid json"})
			return
		}
		if req.Message == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "message is required"})
			return
		}

		start := time.Now()
		reply, err := h.callOpenAIChat(req.Message, req.ChatHistory, req.Model, req.Temperature)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "AI_ERROR", Message: err.Error()})
			return
		}

		result := ChatResult{
			Reply:     reply,
			Model:     req.Model,
			LatencyMS: latency,
		}
		if result.Model == "" {
			var currentModel string
			_ = h.DB.QueryRow(`SELECT model FROM bot_ai_config WHERE id = 1`).Scan(&currentModel)
			if currentModel != "" {
				result.Model = currentModel
			} else {
				result.Model = "gpt-4o-mini"
			}
		}
		result.Usage.PromptTokens = len(req.Message) / 4
		result.Usage.CompletionTokens = len(reply) / 4
		result.Usage.TotalTokens = result.Usage.PromptTokens + result.Usage.CompletionTokens

		_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "ai response generated", Results: result})
	})

	mux.HandleFunc("/bot/ai/tools", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodGet {
			tools := []map[string]any{
				{"name": "send_message", "description": "Send text or media to WhatsApp user"},
				{"name": "manage_group", "description": "Perform group administrative action"},
				{"name": "query_chats", "description": "Query recent chat messages and history"},
				{"name": "update_rules", "description": "Autonomously create or modify bot rules"},
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "tools list", Results: tools})
			return
		}

		if r.Method == http.MethodPost {
			var req ToolRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "invalid json"})
				return
			}
			if req.Tool == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "BAD_REQUEST", Message: "tool name is required"})
				return
			}

			start := time.Now()
			output, err := h.executeTool(req.Tool, req.Parameters)
			latency := time.Since(start).Milliseconds()

			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(APIResponse{Code: "TOOL_ERROR", Message: err.Error()})
				return
			}

			res := ToolResult{
				Tool:      req.Tool,
				Status:    "success",
				LatencyMS: latency,
				Output:    output,
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "tool executed", Results: res})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/bot/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodGet {
			limitStr := r.URL.Query().Get("limit")
			offsetStr := r.URL.Query().Get("offset")
			eventType := r.URL.Query().Get("event_type")
			status := r.URL.Query().Get("status")
			groupJID := r.URL.Query().Get("group_jid")
			search := r.URL.Query().Get("search")

			limit := 50
			if limitStr != "" {
				if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
					limit = l
				}
			}
			offset := 0
			if offsetStr != "" {
				if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
					offset = o
				}
			}

			query := `SELECT id, event_type, rule_id, sender_jid, group_jid, incoming_message, response_message, latency_ms, status, created_at FROM bot_event_logs WHERE 1=1`
			countQuery := `SELECT COUNT(*) FROM bot_event_logs WHERE 1=1`
			var args []any
			var countArgs []any

			if eventType != "" {
				query += ` AND event_type = ?`
				countQuery += ` AND event_type = ?`
				args = append(args, eventType)
				countArgs = append(countArgs, eventType)
			}
			if status != "" {
				query += ` AND status = ?`
				countQuery += ` AND status = ?`
				args = append(args, status)
				countArgs = append(countArgs, status)
			}
			if groupJID != "" {
				query += ` AND group_jid = ?`
				countQuery += ` AND group_jid = ?`
				args = append(args, groupJID)
				countArgs = append(countArgs, groupJID)
			}
			if search != "" {
				query += ` AND (incoming_message LIKE ? OR response_message LIKE ?)`
				countQuery += ` AND (incoming_message LIKE ? OR response_message LIKE ?)`
				args = append(args, "%"+search+"%", "%"+search+"%")
				countArgs = append(countArgs, "%"+search+"%", "%"+search+"%")
			}

			var total int
			_ = h.DB.QueryRow(countQuery, countArgs...).Scan(&total)

			query += ` ORDER BY id DESC LIMIT ? OFFSET ?`
			args = append(args, limit, offset)

			rows, err := h.DB.Query(query, args...)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			logs := make([]BotEventLog, 0)
			for rows.Next() {
				var l BotEventLog
				var cStr string
				if err := rows.Scan(&l.ID, &l.EventType, &l.RuleID, &l.SenderJID, &l.GroupJID, &l.IncomingMessage, &l.ResponseMessage, &l.LatencyMS, &l.Status, &cStr); err == nil {
					l.CreatedAt, _ = time.Parse(time.RFC3339, cStr)
					logs = append(logs, l)
				}
			}

			result := map[string]any{
				"total":  total,
				"limit":  limit,
				"offset": offset,
				"logs":   logs,
			}
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "logs retrieved", Results: result})
			return
		}

		if r.Method == http.MethodDelete {
			res, err := h.DB.Exec(`DELETE FROM bot_event_logs`)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			deleted, _ := res.RowsAffected()
			_ = json.NewEncoder(w).Encode(APIResponse{Code: "SUCCESS", Message: "logs cleared", Results: map[string]any{"deleted_count": deleted}})
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>GOWA Web</title></head><body><div id="root"><nav><span>Bot & Automation</span><a href="#rules">Auto Responder</a><a href="#ai">AI Assistant</a><a href="#groups">Group Moderation</a><a href="#logs">Bot Activity Logs</a></nav></div></body></html>`))
	})

	mux.HandleFunc("/statics/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock-static-content"))
	})

	h.Server = httptest.NewServer(mux)
	h.BaseURL = h.Server.URL
}

func (h *TestHarness) callOpenAIChat(message string, history []ChatMessage, model string, temp *float64) (string, error) {
	var cfg BotAIConfig
	var autoReplyInt int
	var cStr, uStr string
	err := h.DB.QueryRow(`SELECT id, provider, base_url, api_key, model, system_prompt, temperature, trigger_prefix, auto_reply_enabled, created_at, updated_at FROM bot_ai_config WHERE id = 1`).
		Scan(&cfg.ID, &cfg.Provider, &cfg.BaseURL, &cfg.APIKey, &cfg.Model, &cfg.SystemPrompt, &cfg.Temperature, &cfg.TriggerPrefix, &autoReplyInt, &cStr, &uStr)
	if err != nil {
		return "", err
	}

	selectedModel := cfg.Model
	if model != "" {
		selectedModel = model
	}
	selectedTemp := cfg.Temperature
	if temp != nil {
		selectedTemp = *temp
	}

	messages := make([]map[string]any, 0)
	if cfg.SystemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": cfg.SystemPrompt})
	}
	for _, hist := range history {
		messages = append(messages, map[string]any{"role": hist.Role, "content": hist.Content})
	}
	messages = append(messages, map[string]any{"role": "user", "content": message})

	reqBody := map[string]any{
		"model":       selectedModel,
		"messages":    messages,
		"temperature": selectedTemp,
	}
	reqJSON, _ := json.Marshal(reqBody)

	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqJSON))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	resp, err := h.HTTPClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices from openai")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func (h *TestHarness) executeTool(tool string, params map[string]any) (any, error) {
	switch tool {
	case "send_message":
		recipient, _ := params["recipient"].(string)
		msg, _ := params["message"].(string)
		mediaType, _ := params["media_type"].(string)
		mediaURL, _ := params["media_url"].(string)

		if recipient == "" || msg == "" {
			return nil, fmt.Errorf("recipient and message are required for send_message")
		}
		if mediaType == "" {
			mediaType = "text"
		}

		if mediaType == "text" {
			h.Dispatcher.SendText(recipient, msg)
		} else {
			h.Dispatcher.SendMedia(recipient, msg, mediaType, mediaURL)
		}

		return map[string]any{
			"message_id": fmt.Sprintf("MOCK-MSG-%d", time.Now().UnixNano()),
			"recipient":  recipient,
			"status":     "sent",
		}, nil

	case "manage_group":
		action, _ := params["action"].(string)
		groupJID, _ := params["group_jid"].(string)
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
		if rawParts, ok := params["participants"].([]any); ok {
			for _, p := range rawParts {
				if pStr, ok := p.(string); ok {
					participants = append(participants, pStr)
				}
			}
		}

		h.Dispatcher.ManageParticipant(groupJID, action, participants)
		return map[string]any{
			"action":       action,
			"group_jid":    groupJID,
			"participants": participants,
			"status":       "completed",
		}, nil

	case "query_chats":
		action, _ := params["action"].(string)
		if action == "" {
			return nil, fmt.Errorf("action is required for query_chats")
		}
		validChatActions := map[string]bool{
			"list_chats": true, "get_messages": true, "list_contacts": true,
		}
		if !validChatActions[action] {
			return nil, fmt.Errorf("invalid action %q for query_chats", action)
		}

		return map[string]any{
			"action": action,
			"items": []map[string]any{
				{"jid": "user1@s.whatsapp.net", "name": "Alice", "unread": 0},
				{"jid": "group1@g.us", "name": "Dev Team", "unread": 2},
			},
		}, nil

	case "update_rules":
		action, _ := params["action"].(string)
		if action == "" {
			return nil, fmt.Errorf("action is required for update_rules")
		}

		ruleData, _ := params["rule_data"].(map[string]any)
		switch action {
		case "create_rule":
			if ruleData == nil {
				return nil, fmt.Errorf("rule_data is required for create_rule")
			}
			triggerType, _ := ruleData["trigger_type"].(string)
			triggerVal, _ := ruleData["trigger_value"].(string)
			scope, _ := ruleData["scope"].(string)
			respType, _ := ruleData["response_type"].(string)
			respContent, _ := ruleData["response_content"].(string)
			if triggerVal == "" || respContent == "" {
				return nil, fmt.Errorf("trigger_value and response_content are required")
			}
			if scope == "" {
				scope = "all"
			}
			if respType == "" {
				respType = "text"
			}
			res, err := h.DB.Exec(
				`INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content, is_active) VALUES (?, ?, ?, ?, ?, 1)`,
				triggerType, triggerVal, scope, respType, respContent,
			)
			if err != nil {
				return nil, err
			}
			id, _ := res.LastInsertId()
			return map[string]any{"action": "create_rule", "rule_id": id, "status": "created"}, nil

		case "toggle_rule":
			rawID, ok := params["rule_id"]
			if !ok {
				return nil, fmt.Errorf("rule_id is required for toggle_rule")
			}
			ruleID := int64(0)
			switch v := rawID.(type) {
			case float64:
				ruleID = int64(v)
			case int:
				ruleID = int64(v)
			case int64:
				ruleID = v
			}
			_, err := h.DB.Exec(`UPDATE bot_rules SET is_active = CASE WHEN is_active = 1 THEN 0 ELSE 1 END WHERE id = ?`, ruleID)
			if err != nil {
				return nil, err
			}
			return map[string]any{"action": "toggle_rule", "rule_id": ruleID, "status": "toggled"}, nil

		case "delete_rule":
			rawID, ok := params["rule_id"]
			if !ok {
				return nil, fmt.Errorf("rule_id is required for delete_rule")
			}
			ruleID := int64(0)
			switch v := rawID.(type) {
			case float64:
				ruleID = int64(v)
			case int:
				ruleID = int64(v)
			}
			_, err := h.DB.Exec(`DELETE FROM bot_rules WHERE id = ?`, ruleID)
			if err != nil {
				return nil, err
			}
			return map[string]any{"action": "delete_rule", "rule_id": ruleID, "status": "deleted"}, nil

		default:
			return nil, fmt.Errorf("unsupported rule action %q", action)
		}

	default:
		return nil, fmt.Errorf("unsupported tool %q", tool)
	}
}

func (h *TestHarness) MatchRule(incomingText string, isGroup bool) (*BotRule, bool) {
	rows, err := h.DB.Query(`SELECT id, trigger_type, trigger_value, scope, response_type, response_content, media_url, is_active FROM bot_rules WHERE is_active = 1 ORDER BY id ASC`)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	for rows.Next() {
		var rule BotRule
		var isActiveInt int
		if err := rows.Scan(&rule.ID, &rule.TriggerType, &rule.TriggerValue, &rule.Scope, &rule.ResponseType, &rule.ResponseContent, &rule.MediaURL, &isActiveInt); err != nil {
			continue
		}
		rule.IsActive = isActiveInt == 1

		if rule.Scope == "private" && isGroup {
			continue
		}
		if rule.Scope == "group" && !isGroup {
			continue
		}

		matched := false
		lowerIncoming := strings.ToLower(incomingText)
		lowerTrigger := strings.ToLower(rule.TriggerValue)

		switch rule.TriggerType {
		case "exact":
			matched = lowerIncoming == lowerTrigger
		case "contains":
			matched = strings.Contains(lowerIncoming, lowerTrigger)
		case "starts_with":
			matched = strings.HasPrefix(lowerIncoming, lowerTrigger)
		case "regex":
			re, err := regexp.Compile(rule.TriggerValue)
			if err == nil {
				matched = re.MatchString(incomingText)
			}
		}

		if matched {
			return &rule, true
		}
	}
	return nil, false
}

type ModerationResult struct {
	ShouldRevoke bool
	ShouldKick   bool
	Reason       string
}

func (h *TestHarness) ModerateMessage(groupJID string, senderJID string, messageText string) (*ModerationResult, bool) {
	var antiLinkInt int
	err := h.DB.QueryRow(`SELECT anti_link_enabled FROM bot_group_rules WHERE group_jid = ?`, groupJID).Scan(&antiLinkInt)
	if err != nil || antiLinkInt != 1 {
		return nil, false
	}

	linkPatterns := []string{
		`(?i)chat\.whatsapp\.com\/[a-zA-Z0-9]+`,
		`(?i)wa\.me\/[0-9]+`,
		`(?i)https?:\/\/[^\s]+`,
	}

	hasLink := false
	for _, pattern := range linkPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(messageText) {
			hasLink = true
			break
		}
	}

	if hasLink {
		return &ModerationResult{
			ShouldRevoke: true,
			ShouldKick:   false,
			Reason:       "anti_link_violation",
		}, true
	}

	return nil, false
}

func (h *TestHarness) FormatWelcome(template string, userName string, groupName string) string {
	res := strings.ReplaceAll(template, "{name}", userName)
	res = strings.ReplaceAll(res, "{group}", groupName)
	return res
}

func (h *TestHarness) FormatFarewell(template string, userName string, groupName string) string {
	res := strings.ReplaceAll(template, "{name}", userName)
	res = strings.ReplaceAll(res, "{group}", groupName)
	return res
}

func (h *TestHarness) LogEvent(l BotEventLog) error {
	var ruleIDArg any
	if l.RuleID != nil {
		ruleIDArg = *l.RuleID
	}
	_, err := h.DB.Exec(`
		INSERT INTO bot_event_logs (event_type, rule_id, sender_jid, group_jid, incoming_message, response_message, latency_ms, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, l.EventType, ruleIDArg, l.SenderJID, l.GroupJID, l.IncomingMessage, l.ResponseMessage, l.LatencyMS, l.Status)
	return err
}

func (h *TestHarness) DoRequest(method string, path string, body any) (*http.Response, string, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	fullURL := h.BaseURL + path
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return resp, string(respBytes), nil
}

func (h *TestHarness) DoJSON(method string, path string, body any, out any) (int, error) {
	resp, respBody, err := h.DoRequest(method, path, body)
	if err != nil {
		return 0, err
	}
	if out != nil && respBody != "" {
		if err := json.Unmarshal([]byte(respBody), out); err != nil {
			return resp.StatusCode, fmt.Errorf("failed to unmarshal JSON: %w (body: %s)", err, respBody)
		}
	}
	return resp.StatusCode, nil
}
