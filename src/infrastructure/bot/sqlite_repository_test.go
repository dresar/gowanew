package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	domainBot "github.com/dresar/gowanew/domains/bot"
	"github.com/dresar/gowanew/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRepository(t *testing.T) (*SQLiteRepository, *sql.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "bot_test.db")
	connStr := sqlite.FormatChatStorageURI("file:"+dbPath, true, true)

	db, err := sql.Open(sqlite.DriverName, connStr)
	require.NoError(t, err)

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)

	t.Cleanup(func() {
		_ = db.Close()
	})

	repo := NewSQLiteRepository(db).(*SQLiteRepository)
	err = repo.InitializeSchema(context.Background())
	require.NoError(t, err)

	return repo, db
}

func newTestInMemoryRepository(t *testing.T) (*SQLiteRepository, *sql.DB) {
	t.Helper()

	memURI := fmt.Sprintf("file:mem_%d?mode=memory&cache=shared&_pragma=busy_timeout(30000)&_pragma=foreign_keys(1)", time.Now().UnixNano())
	db, err := sql.Open(sqlite.DriverName, memURI)
	require.NoError(t, err)

	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	repo := NewSQLiteRepository(db).(*SQLiteRepository)
	err = repo.InitializeSchema(context.Background())
	require.NoError(t, err)

	return repo, db
}

func TestInitializeSchema_CreatesAllTablesAndIndexes(t *testing.T) {
	_, db := newTestRepository(t)

	expectedTables := []string{
		"bot_schema_info",
		"bot_rules",
		"bot_group_rules",
		"bot_ai_config",
		"bot_event_logs",
	}

	for _, tbl := range expectedTables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", tbl).Scan(&name)
		assert.NoError(t, err)
		assert.Equal(t, tbl, name)
	}

	var journalMode string
	err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	assert.NoError(t, err)
	assert.Equal(t, "wal", journalMode)
}

func TestInitializeSchema_Idempotency(t *testing.T) {
	repo, _ := newTestRepository(t)

	err := repo.InitializeSchema(context.Background())
	assert.NoError(t, err)

	err = repo.InitializeSchema(context.Background())
	assert.NoError(t, err)
}

func TestAIConfig_DefaultRowSeeded(t *testing.T) {
	repo, _ := newTestRepository(t)

	cfg, err := repo.GetAIConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, int64(1), cfg.ID)
	assert.Equal(t, "openai", cfg.Provider)
	assert.Equal(t, "https://api.openai.com/v1", cfg.BaseURL)
	assert.Equal(t, "gpt-4o-mini", cfg.Model)
	assert.Equal(t, 0.7, cfg.Temperature)
	assert.Equal(t, "!ai", cfg.TriggerPrefix)
	assert.False(t, cfg.AutoReplyEnabled)
}

func TestRule_CreateAndGetByID(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	rule := &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "!ping",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "pong",
		MediaURL:        "",
		IsActive:        true,
	}

	created, err := repo.CreateRule(ctx, rule)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Greater(t, created.ID, int64(0))
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())

	fetched, err := repo.GetRuleByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, domainBot.TriggerExact, fetched.TriggerType)
	assert.Equal(t, "!ping", fetched.TriggerValue)
	assert.Equal(t, domainBot.ScopeAll, fetched.Scope)
	assert.Equal(t, "pong", fetched.ResponseContent)
	assert.True(t, fetched.IsActive)
}

func TestRule_GetByID_NotFound(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	fetched, err := repo.GetRuleByID(ctx, 999999)
	assert.Error(t, err)
	assert.Nil(t, fetched)
	assert.True(t, errors.Is(err, domainBot.ErrRuleNotFound) || errors.Is(err, sql.ErrNoRows))
}

func TestRule_ListRules_Filters(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	activeRule := &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "hello",
		Scope:           domainBot.ScopePrivate,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "world",
		IsActive:        true,
	}
	inactiveRule := &domainBot.Rule{
		TriggerType:     domainBot.TriggerContains,
		TriggerValue:    "promo",
		Scope:           domainBot.ScopeGroup,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "discount",
		IsActive:        false,
	}

	_, err := repo.CreateRule(ctx, activeRule)
	require.NoError(t, err)
	_, err = repo.CreateRule(ctx, inactiveRule)
	require.NoError(t, err)

	activeTrue := true
	listActive, err := repo.ListRules(ctx, domainBot.RuleFilter{IsActive: &activeTrue})
	require.NoError(t, err)
	assert.Len(t, listActive, 1)
	assert.Equal(t, "hello", listActive[0].TriggerValue)

	activeFalse := false
	listInactive, err := repo.ListRules(ctx, domainBot.RuleFilter{IsActive: &activeFalse})
	require.NoError(t, err)
	assert.Len(t, listInactive, 1)
	assert.Equal(t, "promo", listInactive[0].TriggerValue)

	scopeGroup := domainBot.ScopeGroup
	listGroup, err := repo.ListRules(ctx, domainBot.RuleFilter{Scope: &scopeGroup})
	require.NoError(t, err)
	assert.Len(t, listGroup, 1)
	assert.Equal(t, "promo", listGroup[0].TriggerValue)

	listSearch, err := repo.ListRules(ctx, domainBot.RuleFilter{Search: "hell"})
	require.NoError(t, err)
	assert.Len(t, listSearch, 1)
	assert.Equal(t, "hello", listSearch[0].TriggerValue)
}

func TestRule_Update(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.CreateRule(ctx, &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "old",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "old_content",
		IsActive:        true,
	})
	require.NoError(t, err)

	newVal := "new_value"
	newContent := "new_content"
	updated, err := repo.UpdateRule(ctx, created.ID, domainBot.UpdateRuleRequest{
		TriggerValue:    &newVal,
		ResponseContent: &newContent,
	})
	require.NoError(t, err)
	assert.Equal(t, "new_value", updated.TriggerValue)
	assert.Equal(t, "new_content", updated.ResponseContent)

	fetched, err := repo.GetRuleByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "new_value", fetched.TriggerValue)
	assert.Equal(t, "new_content", fetched.ResponseContent)
}

func TestRule_ToggleRuleActive(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.CreateRule(ctx, &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "toggle_me",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "content",
		IsActive:        true,
	})
	require.NoError(t, err)
	assert.True(t, created.IsActive)

	toggledOff, err := repo.ToggleRuleActive(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, toggledOff.IsActive)

	toggledOn, err := repo.ToggleRuleActive(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, toggledOn.IsActive)

	toggledAlias, err := repo.ToggleActive(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, toggledAlias.IsActive)
}

func TestRule_Delete(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.CreateRule(ctx, &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "delete_me",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "content",
		IsActive:        true,
	})
	require.NoError(t, err)

	err = repo.DeleteRule(ctx, created.ID)
	require.NoError(t, err)

	fetched, err := repo.GetRuleByID(ctx, created.ID)
	assert.Error(t, err)
	assert.Nil(t, fetched)
}

func TestRule_CheckConstraintViolations(t *testing.T) {
	_, db := newTestRepository(t)

	_, err := db.Exec(`
		INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
		VALUES ('invalid_trigger', 'val', 'all', 'text', 'resp')
	`)
	assert.Error(t, err)

	_, err = db.Exec(`
		INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
		VALUES ('exact', 'val', 'invalid_scope', 'text', 'resp')
	`)
	assert.Error(t, err)
}

func TestGroupRule_UpsertAndGet(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	groupJID := "120363028374928192@g.us"
	initial := &domainBot.GroupRule{
		GroupJID:         groupJID,
		AntiLinkEnabled:  true,
		WelcomeEnabled:   true,
		WelcomeTemplate:  "Welcome {name}!",
		FarewellEnabled:  false,
		FarewellTemplate: "",
	}

	created, err := repo.UpsertGroupRule(ctx, initial)
	require.NoError(t, err)
	assert.Equal(t, groupJID, created.GroupJID)
	assert.True(t, created.AntiLinkEnabled)
	assert.True(t, created.WelcomeEnabled)
	assert.Equal(t, "Welcome {name}!", created.WelcomeTemplate)

	fetched, err := repo.GetGroupRuleByGroupJID(ctx, groupJID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, groupJID, fetched.GroupJID)
	assert.True(t, fetched.AntiLinkEnabled)

	fetchedAlias, err := repo.GetGroupRule(ctx, groupJID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetchedAlias.ID)

	updated := &domainBot.GroupRule{
		GroupJID:         groupJID,
		AntiLinkEnabled:  false,
		WelcomeEnabled:   true,
		WelcomeTemplate:  "Updated welcome!",
		FarewellEnabled:  true,
		FarewellTemplate: "Goodbye {name}!",
	}
	upserted, err := repo.UpsertGroupRule(ctx, updated)
	require.NoError(t, err)
	assert.Equal(t, created.ID, upserted.ID)
	assert.False(t, upserted.AntiLinkEnabled)
	assert.True(t, upserted.FarewellEnabled)

	all, err := repo.ListAllGroupRules(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	allAlias, err := repo.GetGroupRules(ctx)
	require.NoError(t, err)
	assert.Len(t, allAlias, 1)

	err = repo.DeleteGroupRule(ctx, groupJID)
	require.NoError(t, err)

	afterDel, err := repo.GetGroupRuleByGroupJID(ctx, groupJID)
	assert.Error(t, err)
	assert.Nil(t, afterDel)
}

func TestAIConfig_Update(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	newProvider := "openai-compatible"
	newURL := "https://api.groq.com/openai/v1"
	newKey := "gsk_test123456"
	newModel := "llama-3.3-70b-versatile"
	newPrompt := "You are a concise enterprise assistant."
	newTemp := 0.2
	newPrefix := "!bot"
	autoReply := true

	req := domainBot.UpdateAIConfigRequest{
		Provider:         &newProvider,
		BaseURL:          &newURL,
		APIKey:           &newKey,
		Model:            &newModel,
		SystemPrompt:     &newPrompt,
		Temperature:      &newTemp,
		TriggerPrefix:    &newPrefix,
		AutoReplyEnabled: &autoReply,
	}

	updated, err := repo.UpdateAIConfig(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.ID)
	assert.Equal(t, "openai-compatible", updated.Provider)
	assert.Equal(t, "https://api.groq.com/openai/v1", updated.BaseURL)
	assert.Equal(t, "llama-3.3-70b-versatile", updated.Model)
	assert.Equal(t, 0.2, updated.Temperature)
	assert.Equal(t, "!bot", updated.TriggerPrefix)
	assert.True(t, updated.AutoReplyEnabled)

	fetched, err := repo.GetAIConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "llama-3.3-70b-versatile", fetched.Model)
	assert.True(t, fetched.AutoReplyEnabled)
}

func TestAIConfig_SingletonConstraint(t *testing.T) {
	_, db := newTestRepository(t)

	_, err := db.Exec(`
		INSERT INTO bot_ai_config (id, provider, base_url, api_key, model)
		VALUES (2, 'openai', 'https://api.openai.com/v1', '', 'gpt-4o')
	`)
	assert.Error(t, err)
}

func TestEventLog_CreateAndListWithPagination(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	for i := 1; i <= 15; i++ {
		evType := domainBot.EventTypeAutoReply
		if i%3 == 0 {
			evType = domainBot.EventTypeAIChat
		}
		status := domainBot.LogStatusSuccess
		if i%5 == 0 {
			status = domainBot.LogStatusFailed
		}

		log := &domainBot.EventLog{
			EventType:       evType,
			RuleID:          nil,
			SenderJID:       fmt.Sprintf("62812000000%02d@s.whatsapp.net", i),
			GroupJID:        "",
			IncomingMessage: fmt.Sprintf("message %d", i),
			ResponseMessage: fmt.Sprintf("reply %d", i),
			LatencyMS:       int64(10 * i),
			Status:          status,
		}
		_, err := repo.CreateEventLog(ctx, log)
		require.NoError(t, err)
	}

	logsPage1, total, err := repo.ListEventLogs(ctx, domainBot.EventLogFilter{
		Limit:  5,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(15), total)
	assert.Len(t, logsPage1, 5)

	logsPage2, total2, err := repo.ListEventLogs(ctx, domainBot.EventLogFilter{
		Limit:  5,
		Offset: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(15), total2)
	assert.Len(t, logsPage2, 5)
	assert.NotEqual(t, logsPage1[0].ID, logsPage2[0].ID)

	filterType := domainBot.EventTypeAIChat
	logsAI, totalAI, err := repo.ListEventLogs(ctx, domainBot.EventLogFilter{
		EventType: &filterType,
		Limit:     10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), totalAI)
	assert.Len(t, logsAI, 5)

	purged, err := repo.PurgeEventLogs(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(15), purged)

	_, totalAfter, err := repo.ListEventLogs(ctx, domainBot.EventLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), totalAfter)
}

func TestConcurrentWritersAndReaders_NoLockErrors(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	const numGoroutines = 20
	const opsPerGoroutine = 10

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*opsPerGoroutine*3)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				rule := &domainBot.Rule{
					TriggerType:     domainBot.TriggerExact,
					TriggerValue:    fmt.Sprintf("w%d_rule_%d", workerID, i),
					Scope:           domainBot.ScopeAll,
					ResponseType:    domainBot.ResponseTypeText,
					ResponseContent: fmt.Sprintf("resp_%d", i),
					IsActive:        true,
				}
				if _, err := repo.CreateRule(ctx, rule); err != nil {
					errCh <- fmt.Errorf("create rule error worker %d: %w", workerID, err)
				}

				groupRule := &domainBot.GroupRule{
					GroupJID:        fmt.Sprintf("12036300%02d@g.us", workerID),
					AntiLinkEnabled: i%2 == 0,
					WelcomeEnabled:  true,
					WelcomeTemplate: "welcome",
				}
				if _, err := repo.UpsertGroupRule(ctx, groupRule); err != nil {
					errCh <- fmt.Errorf("upsert group error worker %d: %w", workerID, err)
				}

				log := &domainBot.EventLog{
					EventType:       domainBot.EventTypeAutoReply,
					SenderJID:       fmt.Sprintf("user_%d@s.whatsapp.net", workerID),
					IncomingMessage: "ping",
					ResponseMessage: "pong",
					LatencyMS:       15,
					Status:          domainBot.LogStatusSuccess,
				}
				if _, err := repo.CreateEventLog(ctx, log); err != nil {
					errCh <- fmt.Errorf("create log error worker %d: %w", workerID, err)
				}

				if _, err := repo.ListRules(ctx, domainBot.RuleFilter{Limit: 5}); err != nil {
					errCh <- fmt.Errorf("read rules error worker %d: %w", workerID, err)
				}
				if _, err := repo.GetAIConfig(ctx); err != nil {
					errCh <- fmt.Errorf("read ai config error worker %d: %w", workerID, err)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrency failure: %v", err)
	}

	rules, err := repo.ListRules(ctx, domainBot.RuleFilter{Limit: 500})
	require.NoError(t, err)
	assert.Equal(t, numGoroutines*opsPerGoroutine, len(rules))
}

func TestContextCancellation(t *testing.T) {
	repo, _ := newTestRepository(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.GetRuleByID(ctx, 1)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "canceled"))
}

func TestRepository_InMemoryMode(t *testing.T) {
	repo, _ := newTestInMemoryRepository(t)
	ctx := context.Background()

	rule, err := repo.CreateRule(ctx, &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "!inmem",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "in_memory_ok",
		IsActive:        true,
	})
	require.NoError(t, err)
	assert.Greater(t, rule.ID, int64(0))

	fetched, err := repo.GetRuleByID(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, "!inmem", fetched.TriggerValue)

	cfg, err := repo.GetAIConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "openai", cfg.Provider)
}

func TestNokomenCompliance(t *testing.T) {
	dirs := []string{".", "../../domains/bot"}
	fset := token.NewFileSet()
	var violations []string

	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		require.NoError(t, err)

		for _, file := range files {
			node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
			require.NoError(t, err, "failed to parse %s", file)

			for _, cg := range node.Comments {
				for _, comment := range cg.List {
					text := strings.TrimSpace(comment.Text)
					if strings.HasPrefix(text, "//go:build") {
						continue
					}
					violations = append(violations, fmt.Sprintf("%s:%d: %s", file, fset.Position(comment.Pos()).Line, text))
				}
			}
		}
	}

	assert.Empty(t, violations, "nokomen violation: comments found in Go code:\n%s", strings.Join(violations, "\n"))
}
