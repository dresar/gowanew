package bot

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainBot "github.com/dresar/gowanew/domains/bot"
	"github.com/dresar/gowanew/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdversarial_FaultTolerance_CancelledContext(t *testing.T) {
	repo, _ := newTestRepository(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.CreateRule(ctx, &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "!test",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "resp",
	})
	assert.Error(t, err)

	_, err = repo.GetRuleByID(ctx, 1)
	assert.Error(t, err)

	_, err = repo.ListRules(ctx, domainBot.RuleFilter{})
	assert.Error(t, err)

	val := "val"
	_, err = repo.UpdateRule(ctx, 1, domainBot.UpdateRuleRequest{TriggerValue: &val})
	assert.Error(t, err)

	err = repo.DeleteRule(ctx, 1)
	assert.Error(t, err)

	_, err = repo.ToggleRuleActive(ctx, 1)
	assert.Error(t, err)

	_, err = repo.GetGroupRuleByGroupJID(ctx, "group@g.us")
	assert.Error(t, err)

	_, err = repo.ListAllGroupRules(ctx)
	assert.Error(t, err)

	_, err = repo.UpsertGroupRule(ctx, &domainBot.GroupRule{GroupJID: "group@g.us"})
	assert.Error(t, err)

	err = repo.DeleteGroupRule(ctx, "group@g.us")
	assert.Error(t, err)

	_, err = repo.GetAIConfig(ctx)
	assert.Error(t, err)

	prov := "openai"
	_, err = repo.UpdateAIConfig(ctx, domainBot.UpdateAIConfigRequest{Provider: &prov})
	assert.Error(t, err)

	_, err = repo.CreateEventLog(ctx, &domainBot.EventLog{
		EventType:       domainBot.EventTypeAutoReply,
		SenderJID:       "user@s.whatsapp.net",
		Status:          domainBot.LogStatusSuccess,
		IncomingMessage: "hi",
		ResponseMessage: "hello",
	})
	assert.Error(t, err)

	_, _, err = repo.ListEventLogs(ctx, domainBot.EventLogFilter{})
	assert.Error(t, err)

	_, err = repo.PurgeEventLogs(ctx, nil)
	assert.Error(t, err)

	err = repo.InitializeSchema(ctx)
	assert.Error(t, err)

	liveCtx := context.Background()
	created, err := repo.CreateRule(liveCtx, &domainBot.Rule{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "!alive",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "alive",
		IsActive:        true,
	})
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
}

func TestAdversarial_FaultTolerance_QueryTimeout(t *testing.T) {
	repo, _ := newTestRepository(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	_, err := repo.ListRules(ctx, domainBot.RuleFilter{Limit: 10})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "context") || strings.Contains(err.Error(), "canceled") || strings.Contains(err.Error(), "deadline"))

	liveCtx := context.Background()
	rules, err := repo.ListRules(liveCtx, domainBot.RuleFilter{Limit: 10})
	assert.NoError(t, err)
	assert.NotNil(t, rules)
}

func TestAdversarial_FaultTolerance_InvalidDatabasePaths(t *testing.T) {
	tempDir := t.TempDir()
	filePathAsDir := filepath.Join(tempDir, "blocking_file")
	err := os.WriteFile(filePathAsDir, []byte("blocker"), 0644)
	require.NoError(t, err)

	impossiblePath := filepath.Join(filePathAsDir, "child_folder", "bot.db")
	db, err := OpenBotDB("file:"+impossiblePath, 1)
	assert.Error(t, err)
	assert.Nil(t, db)

	dbZero, err := OpenBotDB(filepath.Join(tempDir, "zero_conns.db"), 0)
	require.NoError(t, err)
	require.NotNil(t, dbZero)
	_ = dbZero.Close()

	dbNeg, err := OpenBotDB(filepath.Join(tempDir, "neg_conns.db"), -5)
	require.NoError(t, err)
	require.NotNil(t, dbNeg)
	_ = dbNeg.Close()
}

func TestAdversarial_FaultTolerance_TransactionRollback(t *testing.T) {
	_, db := newTestRepository(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO bot_rules (
			trigger_type, trigger_value, scope, response_type,
			response_content, media_url, is_active, created_at, updated_at
		) VALUES ('exact', '!rollback_test', 'all', 'text', 'will_rollback', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
		VALUES ('invalid_trigger', 'bad', 'all', 'text', 'bad')
	`)
	assert.Error(t, err)

	err = tx.Rollback()
	require.NoError(t, err)

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bot_rules WHERE trigger_value = '!rollback_test'").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestAdversarial_FaultTolerance_MigrationRollback(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migration_fail.db")
	connStr := sqlite.FormatChatStorageURI("file:"+dbPath, true, true)
	db, err := sql.Open(sqlite.DriverName, connStr)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	_, err = db.ExecContext(ctx, "CREATE TABLE bot_schema_info (version INTEGER PRIMARY KEY, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "CREATE TABLE bot_rules (id TEXT PRIMARY KEY)")
	require.NoError(t, err)

	err = RunMigrations(ctx, db)
	assert.Error(t, err)

	var recordedCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bot_schema_info WHERE version = 2").Scan(&recordedCount)
	require.NoError(t, err)
	assert.Equal(t, 0, recordedCount)

	var indexCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_bot_rules_is_active'").Scan(&indexCount)
	require.NoError(t, err)
	assert.Equal(t, 0, indexCount)
}

func TestAdversarial_SchemaConstraints_InvalidTriggerTypes(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()

	invalidTriggers := []string{"", "invalid", "EXACT", "wildcard", "prefix", "suffix", "123"}
	for _, bad := range invalidTriggers {
		_, err := repo.CreateRule(ctx, &domainBot.Rule{
			TriggerType:     domainBot.TriggerType(bad),
			TriggerValue:    "test",
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "test",
			IsActive:        true,
		})
		assert.Errorf(t, err, "expected error for trigger_type: %s", bad)

		_, err = db.ExecContext(ctx, `
			INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
			VALUES (?, 'test', 'all', 'text', 'test')
		`, bad)
		assert.Errorf(t, err, "expected raw SQL error for trigger_type: %s", bad)
	}

	var totalRules int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bot_rules").Scan(&totalRules)
	require.NoError(t, err)
	assert.Equal(t, 0, totalRules)
}

func TestAdversarial_SchemaConstraints_InvalidScopes(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()

	invalidScopes := []string{"", "invalid", "ALL", "channel", "broadcast", "user", "global"}
	for _, bad := range invalidScopes {
		_, err := repo.CreateRule(ctx, &domainBot.Rule{
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    "test",
			Scope:           domainBot.Scope(bad),
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "test",
			IsActive:        true,
		})
		assert.Errorf(t, err, "expected error for scope: %s", bad)

		_, err = db.ExecContext(ctx, `
			INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
			VALUES ('exact', 'test', ?, 'text', 'test')
		`, bad)
		assert.Errorf(t, err, "expected raw SQL error for scope: %s", bad)
	}

	var totalRules int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bot_rules").Scan(&totalRules)
	require.NoError(t, err)
	assert.Equal(t, 0, totalRules)
}

func TestAdversarial_SchemaConstraints_InvalidResponseTypes(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()

	invalidTypes := []string{"", "audio", "video", "TEXT", "document", "json"}
	for _, bad := range invalidTypes {
		_, err := repo.CreateRule(ctx, &domainBot.Rule{
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    "test",
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseType(bad),
			ResponseContent: "test",
			IsActive:        true,
		})
		assert.Errorf(t, err, "expected error for response_type: %s", bad)

		_, err = db.ExecContext(ctx, `
			INSERT INTO bot_rules (trigger_type, trigger_value, scope, response_type, response_content)
			VALUES ('exact', 'test', 'all', ?, 'test')
		`, bad)
		assert.Errorf(t, err, "expected raw SQL error for response_type: %s", bad)
	}
}

func TestAdversarial_SchemaConstraints_DuplicateGroupJID(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()

	groupJID := "120363099999999999@g.us"

	_, err := db.ExecContext(ctx, `
		INSERT INTO bot_group_rules (group_jid, anti_link_enabled)
		VALUES (?, 0)
	`, groupJID)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO bot_group_rules (group_jid, anti_link_enabled)
		VALUES (?, 1)
	`, groupJID)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "constraint"))

	upserted, err := repo.UpsertGroupRule(ctx, &domainBot.GroupRule{
		GroupJID:        groupJID,
		AntiLinkEnabled: true,
		WelcomeEnabled:  true,
		WelcomeTemplate: "Hello new member!",
	})
	require.NoError(t, err)
	assert.True(t, upserted.AntiLinkEnabled)
	assert.True(t, upserted.WelcomeEnabled)

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bot_group_rules WHERE group_jid = ?", groupJID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAdversarial_SchemaConstraints_MultipleAIConfigRows(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()

	cfg, err := repo.GetAIConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cfg.ID)

	badIDs := []int{2, 0, -1, 99}
	for _, badID := range badIDs {
		_, err := db.ExecContext(ctx, `
			INSERT INTO bot_ai_config (id, provider, base_url, api_key, model)
			VALUES (?, 'openai', 'https://api.openai.com/v1', '', 'gpt-4o')
		`, badID)
		assert.Errorf(t, err, "expected error for bot_ai_config with id: %d", badID)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO bot_ai_config (provider, base_url, api_key, model)
		VALUES ('openai', 'https://api.openai.com/v1', '', 'gpt-4o')
	`)
	assert.Error(t, err)

	newModel := "gpt-4o"
	updated, err := repo.UpdateAIConfig(ctx, domainBot.UpdateAIConfigRequest{Model: &newModel})
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated.ID)
	assert.Equal(t, "gpt-4o", updated.Model)

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bot_ai_config").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestAdversarial_SchemaConstraints_EventLogTypesAndStatus(t *testing.T) {
	repo, db := newTestRepository(t)
	ctx := context.Background()

	badEventTypes := []string{"", "unknown", "AUTO_REPLY", "debug", "broadcast"}
	for _, badType := range badEventTypes {
		_, err := repo.CreateEventLog(ctx, &domainBot.EventLog{
			EventType:       domainBot.EventType(badType),
			SenderJID:       "user@s.whatsapp.net",
			IncomingMessage: "msg",
			ResponseMessage: "resp",
			Status:          domainBot.LogStatusSuccess,
		})
		assert.Errorf(t, err, "expected error for bad event_type: %s", badType)

		_, err = db.ExecContext(ctx, `
			INSERT INTO bot_event_logs (event_type, sender_jid, status)
			VALUES (?, 'user@s.whatsapp.net', 'success')
		`, badType)
		assert.Errorf(t, err, "expected raw SQL error for bad event_type: %s", badType)
	}

	badStatuses := []string{"", "pending", "SUCCESS", "timeout", "queued"}
	for _, badStat := range badStatuses {
		_, err := repo.CreateEventLog(ctx, &domainBot.EventLog{
			EventType:       domainBot.EventTypeAutoReply,
			SenderJID:       "user@s.whatsapp.net",
			IncomingMessage: "msg",
			ResponseMessage: "resp",
			Status:          domainBot.LogStatus(badStat),
		})
		assert.Errorf(t, err, "expected error for bad status: %s", badStat)

		_, err = db.ExecContext(ctx, `
			INSERT INTO bot_event_logs (event_type, sender_jid, status)
			VALUES ('auto_reply', 'user@s.whatsapp.net', ?)
		`, badStat)
		assert.Errorf(t, err, "expected raw SQL error for bad status: %s", badStat)
	}

	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM bot_event_logs").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestAdversarial_SchemaConstraints_BooleanCheckConstraints(t *testing.T) {
	_, db := newTestRepository(t)
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `
		INSERT INTO bot_rules (
			trigger_type, trigger_value, scope, response_type, response_content, is_active
		) VALUES ('exact', '!test', 'all', 'text', 'resp', 2)
	`)
	assert.Error(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO bot_group_rules (group_jid, anti_link_enabled)
		VALUES ('test@g.us', -1)
	`)
	assert.Error(t, err)

	_, err = db.ExecContext(ctx, `
		UPDATE bot_ai_config SET auto_reply_enabled = 5 WHERE id = 1
	`)
	assert.Error(t, err)
}
