package bot

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	domainBot "github.com/dresar/gowanew/domains/bot"
	"github.com/dresar/gowanew/pkg/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
	mu sync.RWMutex
}

func OpenBotDB(uri string, maxConns int) (*sql.DB, error) {
	cleanURI := uri
	if strings.HasPrefix(cleanURI, "file:") {
		cleanURI = strings.TrimPrefix(cleanURI, "file:")
	}
	if qIdx := strings.Index(cleanURI, "?"); qIdx != -1 {
		cleanURI = cleanURI[:qIdx]
	}
	if !strings.HasPrefix(cleanURI, ":memory:") && !strings.Contains(cleanURI, "mode=memory") {
		dbDir := filepath.Dir(cleanURI)
		if dbDir != "" && dbDir != "." {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				return nil, fmt.Errorf("create bot storage dir: %w", err)
			}
		}
	}

	connStr := sqlite.FormatChatStorageURI(uri, true, true)
	db, err := sql.Open(sqlite.DriverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("open bot sqlite: %w", err)
	}

	if maxConns < 1 {
		maxConns = 1
	}
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping bot sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 30000;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, p := range pragmas {
		_, _ = db.ExecContext(ctx, p)
	}

	return db, nil
}

func NewSQLiteRepository(db *sql.DB) domainBot.IBotRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) InitializeSchema(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return RunMigrations(ctx, r.db)
}

func (r *SQLiteRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func parseTime(val any) time.Time {
	switch v := val.(type) {
	case time.Time:
		return v
	case []byte:
		return parseTimeString(string(v))
	case string:
		return parseTimeString(v)
	case int64:
		return time.Unix(v, 0).UTC()
	default:
		return time.Time{}
	}
}

func parseTimeString(s string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (r *SQLiteRepository) scanRule(scanner interface{ Scan(...any) error }) (*domainBot.Rule, error) {
	rule := &domainBot.Rule{}
	var isActiveInt int
	var rawCreated, rawUpdated any
	err := scanner.Scan(
		&rule.ID,
		&rule.TriggerType,
		&rule.TriggerValue,
		&rule.RecipientJID,
		&rule.Scope,
		&rule.ResponseType,
		&rule.ResponseContent,
		&rule.MediaURL,
		&isActiveInt,
		&rawCreated,
		&rawUpdated,
	)
	if err != nil {
		return nil, err
	}
	rule.IsActive = isActiveInt == 1
	rule.CreatedAt = parseTime(rawCreated)
	rule.UpdatedAt = parseTime(rawUpdated)
	return rule, nil
}

func (r *SQLiteRepository) scanGroupRule(scanner interface{ Scan(...any) error }) (*domainBot.GroupRule, error) {
	rule := &domainBot.GroupRule{}
	var antiLinkInt, welcomeInt, farewellInt int
	var rawCreated, rawUpdated any
	err := scanner.Scan(
		&rule.ID,
		&rule.GroupJID,
		&antiLinkInt,
		&welcomeInt,
		&rule.WelcomeTemplate,
		&farewellInt,
		&rule.FarewellTemplate,
		&rawCreated,
		&rawUpdated,
	)
	if err != nil {
		return nil, err
	}
	rule.AntiLinkEnabled = antiLinkInt == 1
	rule.WelcomeEnabled = welcomeInt == 1
	rule.FarewellEnabled = farewellInt == 1
	rule.CreatedAt = parseTime(rawCreated)
	rule.UpdatedAt = parseTime(rawUpdated)
	return rule, nil
}

func (r *SQLiteRepository) scanAIConfig(scanner interface{ Scan(...any) error }) (*domainBot.AIConfig, error) {
	cfg := &domainBot.AIConfig{}
	var autoReplyInt int
	var rawCreated, rawUpdated any
	err := scanner.Scan(
		&cfg.ID,
		&cfg.Provider,
		&cfg.BaseURL,
		&cfg.APIKey,
		&cfg.Model,
		&cfg.SystemPrompt,
		&cfg.Temperature,
		&cfg.TriggerPrefix,
		&autoReplyInt,
		&rawCreated,
		&rawUpdated,
	)
	if err != nil {
		return nil, err
	}
	cfg.AutoReplyEnabled = autoReplyInt == 1
	cfg.CreatedAt = parseTime(rawCreated)
	cfg.UpdatedAt = parseTime(rawUpdated)
	return cfg, nil
}

func (r *SQLiteRepository) scanEventLog(scanner interface{ Scan(...any) error }) (*domainBot.EventLog, error) {
	log := &domainBot.EventLog{}
	var ruleID sql.NullInt64
	var rawCreated any
	err := scanner.Scan(
		&log.ID,
		&log.EventType,
		&ruleID,
		&log.SenderJID,
		&log.GroupJID,
		&log.IncomingMessage,
		&log.ResponseMessage,
		&log.LatencyMS,
		&log.Status,
		&rawCreated,
	)
	if err != nil {
		return nil, err
	}
	if ruleID.Valid {
		v := ruleID.Int64
		log.RuleID = &v
	}
	log.CreatedAt = parseTime(rawCreated)
	return log, nil
}

func (r *SQLiteRepository) CreateRule(ctx context.Context, rule *domainBot.Rule) (*domainBot.Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	isActiveInt := 1
	if !rule.IsActive {
		isActiveInt = 0
	}

	query := `
		INSERT INTO bot_rules (
			trigger_type, trigger_value, recipient_jid, scope, response_type,
			response_content, media_url, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := r.db.ExecContext(ctx, query,
		string(rule.TriggerType),
		rule.TriggerValue,
		rule.RecipientJID,
		string(rule.Scope),
		string(rule.ResponseType),
		rule.ResponseContent,
		rule.MediaURL,
		isActiveInt,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	rule.ID = id
	return rule, nil
}

func (r *SQLiteRepository) GetRuleByID(ctx context.Context, id int64) (*domainBot.Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, trigger_type, trigger_value, recipient_jid, scope, response_type,
			   response_content, media_url, is_active, created_at, updated_at
		FROM bot_rules
		WHERE id = ?
		LIMIT 1
	`

	rule, err := r.scanRule(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, domainBot.ErrRuleNotFound
	}
	return rule, err
}

func (r *SQLiteRepository) ListRules(ctx context.Context, filter domainBot.RuleFilter) ([]*domainBot.Rule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, trigger_type, trigger_value, recipient_jid, scope, response_type,
			   response_content, media_url, is_active, created_at, updated_at
		FROM bot_rules
	`
	var conditions []string
	var args []any

	if filter.IsActive != nil {
		activeInt := 0
		if *filter.IsActive {
			activeInt = 1
		}
		conditions = append(conditions, "is_active = ?")
		args = append(args, activeInt)
	}

	if filter.Scope != nil && *filter.Scope != "" {
		conditions = append(conditions, "scope = ?")
		args = append(args, string(*filter.Scope))
	}

	if filter.RecipientJID != nil && *filter.RecipientJID != "" {
		if *filter.RecipientJID == "global" {
			conditions = append(conditions, "(recipient_jid = '' OR recipient_jid IS NULL)")
		} else {
			conditions = append(conditions, "recipient_jid LIKE ?")
			args = append(args, "%"+*filter.RecipientJID+"%")
		}
	}

	if filter.Search != "" {
		conditions = append(conditions, "(trigger_value LIKE ? OR response_content LIKE ? OR recipient_jid LIKE ?)")
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern, pattern, pattern)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY id DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]*domainBot.Rule, 0)
	for rows.Next() {
		rule, err := r.scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *SQLiteRepository) UpdateRule(ctx context.Context, id int64, req domainBot.UpdateRuleRequest) (*domainBot.Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.scanRule(r.db.QueryRowContext(ctx, `
		SELECT id, trigger_type, trigger_value, recipient_jid, scope, response_type,
			   response_content, media_url, is_active, created_at, updated_at
		FROM bot_rules WHERE id = ? LIMIT 1
	`, id))
	if err == sql.ErrNoRows {
		return nil, domainBot.ErrRuleNotFound
	}
	if err != nil {
		return nil, err
	}

	if req.TriggerType != nil {
		existing.TriggerType = *req.TriggerType
	}
	if req.TriggerValue != nil {
		existing.TriggerValue = *req.TriggerValue
	}
	if req.RecipientJID != nil {
		existing.RecipientJID = *req.RecipientJID
	}
	if req.Scope != nil {
		existing.Scope = *req.Scope
	}
	if req.ResponseType != nil {
		existing.ResponseType = *req.ResponseType
	}
	if req.ResponseContent != nil {
		existing.ResponseContent = *req.ResponseContent
	}
	if req.MediaURL != nil {
		existing.MediaURL = *req.MediaURL
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	existing.UpdatedAt = time.Now().UTC()

	activeInt := 0
	if existing.IsActive {
		activeInt = 1
	}

	query := `
		UPDATE bot_rules
		SET trigger_type = ?, trigger_value = ?, recipient_jid = ?, scope = ?, response_type = ?,
			response_content = ?, media_url = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = r.db.ExecContext(ctx, query,
		string(existing.TriggerType),
		existing.TriggerValue,
		existing.RecipientJID,
		string(existing.Scope),
		string(existing.ResponseType),
		existing.ResponseContent,
		existing.MediaURL,
		activeInt,
		existing.UpdatedAt,
		id,
	)
	if err != nil {
		return nil, err
	}
	return existing, nil
}

func (r *SQLiteRepository) DeleteRule(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.ExecContext(ctx, "DELETE FROM bot_rules WHERE id = ?", id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domainBot.ErrRuleNotFound
	}
	return nil
}

func (r *SQLiteRepository) ToggleRuleActive(ctx context.Context, id int64) (*domainBot.Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	query := `
		UPDATE bot_rules
		SET is_active = CASE WHEN is_active = 1 THEN 0 ELSE 1 END,
			updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, domainBot.ErrRuleNotFound
	}

	selectQuery := `
		SELECT id, trigger_type, trigger_value, scope, response_type,
			   response_content, media_url, is_active, created_at, updated_at
		FROM bot_rules
		WHERE id = ?
		LIMIT 1
	`
	return r.scanRule(r.db.QueryRowContext(ctx, selectQuery, id))
}

func (r *SQLiteRepository) ToggleActive(ctx context.Context, id int64) (*domainBot.Rule, error) {
	return r.ToggleRuleActive(ctx, id)
}

func (r *SQLiteRepository) GetGroupRuleByGroupJID(ctx context.Context, groupJID string) (*domainBot.GroupRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, group_jid, anti_link_enabled, welcome_enabled, welcome_template,
			   farewell_enabled, farewell_template, created_at, updated_at
		FROM bot_group_rules
		WHERE group_jid = ?
		LIMIT 1
	`
	rule, err := r.scanGroupRule(r.db.QueryRowContext(ctx, query, groupJID))
	if err == sql.ErrNoRows {
		return nil, domainBot.ErrGroupRuleNotFound
	}
	return rule, err
}

func (r *SQLiteRepository) GetGroupRule(ctx context.Context, groupJID string) (*domainBot.GroupRule, error) {
	return r.GetGroupRuleByGroupJID(ctx, groupJID)
}

func (r *SQLiteRepository) ListAllGroupRules(ctx context.Context) ([]*domainBot.GroupRule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, group_jid, anti_link_enabled, welcome_enabled, welcome_template,
			   farewell_enabled, farewell_template, created_at, updated_at
		FROM bot_group_rules
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]*domainBot.GroupRule, 0)
	for rows.Next() {
		rule, err := r.scanGroupRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *SQLiteRepository) GetGroupRules(ctx context.Context) ([]*domainBot.GroupRule, error) {
	return r.ListAllGroupRules(ctx)
}

func (r *SQLiteRepository) UpsertGroupRule(ctx context.Context, rule *domainBot.GroupRule) (*domainBot.GroupRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	antiLinkInt := 0
	if rule.AntiLinkEnabled {
		antiLinkInt = 1
	}
	welcomeInt := 0
	if rule.WelcomeEnabled {
		welcomeInt = 1
	}
	farewellInt := 0
	if rule.FarewellEnabled {
		farewellInt = 1
	}

	query := `
		INSERT INTO bot_group_rules (
			group_jid, anti_link_enabled, welcome_enabled, welcome_template,
			farewell_enabled, farewell_template, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(group_jid) DO UPDATE SET
			anti_link_enabled = excluded.anti_link_enabled,
			welcome_enabled = excluded.welcome_enabled,
			welcome_template = excluded.welcome_template,
			farewell_enabled = excluded.farewell_enabled,
			farewell_template = excluded.farewell_template,
			updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(ctx, query,
		rule.GroupJID,
		antiLinkInt,
		welcomeInt,
		rule.WelcomeTemplate,
		farewellInt,
		rule.FarewellTemplate,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	selectQuery := `
		SELECT id, group_jid, anti_link_enabled, welcome_enabled, welcome_template,
			   farewell_enabled, farewell_template, created_at, updated_at
		FROM bot_group_rules
		WHERE group_jid = ?
		LIMIT 1
	`
	return r.scanGroupRule(r.db.QueryRowContext(ctx, selectQuery, rule.GroupJID))
}

func (r *SQLiteRepository) DeleteGroupRule(ctx context.Context, groupJID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.ExecContext(ctx, "DELETE FROM bot_group_rules WHERE group_jid = ?", groupJID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domainBot.ErrGroupRuleNotFound
	}
	return nil
}

func (r *SQLiteRepository) GetAIConfig(ctx context.Context) (*domainBot.AIConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := `
		SELECT id, provider, base_url, api_key, model,
			   system_prompt, temperature, trigger_prefix, auto_reply_enabled,
			   created_at, updated_at
		FROM bot_ai_config
		WHERE id = 1
		LIMIT 1
	`
	cfg, err := r.scanAIConfig(r.db.QueryRowContext(ctx, query))
	if err == sql.ErrNoRows {
		return nil, domainBot.ErrAIConfigNotFound
	}
	return cfg, err
}

func (r *SQLiteRepository) UpdateAIConfig(ctx context.Context, req domainBot.UpdateAIConfigRequest) (*domainBot.AIConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.scanAIConfig(r.db.QueryRowContext(ctx, `
		SELECT id, provider, base_url, api_key, model,
			   system_prompt, temperature, trigger_prefix, auto_reply_enabled,
			   created_at, updated_at
		FROM bot_ai_config
		WHERE id = 1
		LIMIT 1
	`))
	if err == sql.ErrNoRows {
		now := time.Now().UTC()
		existing = &domainBot.AIConfig{
			ID:               1,
			Provider:         "openai",
			BaseURL:          "https://api.openai.com/v1",
			APIKey:           "",
			Model:            "gpt-4o-mini",
			SystemPrompt:     "",
			Temperature:      0.7,
			TriggerPrefix:    "!ai",
			AutoReplyEnabled: false,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
	} else if err != nil {
		return nil, err
	}

	if req.Provider != nil {
		existing.Provider = *req.Provider
	}
	if req.BaseURL != nil {
		existing.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		existing.APIKey = *req.APIKey
	}
	if req.Model != nil {
		existing.Model = *req.Model
	}
	if req.SystemPrompt != nil {
		existing.SystemPrompt = *req.SystemPrompt
	}
	if req.Temperature != nil {
		existing.Temperature = *req.Temperature
	}
	if req.TriggerPrefix != nil {
		existing.TriggerPrefix = *req.TriggerPrefix
	}
	if req.AutoReplyEnabled != nil {
		existing.AutoReplyEnabled = *req.AutoReplyEnabled
	}
	existing.UpdatedAt = time.Now().UTC()

	autoReplyInt := 0
	if existing.AutoReplyEnabled {
		autoReplyInt = 1
	}

	query := `
		INSERT INTO bot_ai_config (
			id, provider, base_url, api_key, model, system_prompt,
			temperature, trigger_prefix, auto_reply_enabled, created_at, updated_at
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider = excluded.provider,
			base_url = excluded.base_url,
			api_key = excluded.api_key,
			model = excluded.model,
			system_prompt = excluded.system_prompt,
			temperature = excluded.temperature,
			trigger_prefix = excluded.trigger_prefix,
			auto_reply_enabled = excluded.auto_reply_enabled,
			updated_at = excluded.updated_at
	`
	_, err = r.db.ExecContext(ctx, query,
		existing.Provider,
		existing.BaseURL,
		existing.APIKey,
		existing.Model,
		existing.SystemPrompt,
		existing.Temperature,
		existing.TriggerPrefix,
		autoReplyInt,
		existing.CreatedAt,
		existing.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return existing, nil
}

func (r *SQLiteRepository) CreateEventLog(ctx context.Context, log *domainBot.EventLog) (*domainBot.EventLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	var ruleIDVal any
	if log.RuleID != nil {
		ruleIDVal = *log.RuleID
	}

	query := `
		INSERT INTO bot_event_logs (
			event_type, rule_id, sender_jid, group_jid,
			incoming_message, response_message, latency_ms, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query,
		string(log.EventType),
		ruleIDVal,
		log.SenderJID,
		log.GroupJID,
		log.IncomingMessage,
		log.ResponseMessage,
		log.LatencyMS,
		string(log.Status),
		log.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	log.ID = id
	return log, nil
}

func (r *SQLiteRepository) CreateLog(ctx context.Context, log *domainBot.EventLog) (*domainBot.EventLog, error) {
	return r.CreateEventLog(ctx, log)
}

func (r *SQLiteRepository) ListEventLogs(ctx context.Context, filter domainBot.EventLogFilter) ([]*domainBot.EventLog, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var conditions []string
	var args []any

	if filter.EventType != nil && *filter.EventType != "" {
		conditions = append(conditions, "event_type = ?")
		args = append(args, string(*filter.EventType))
	}
	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.GroupJID != "" {
		conditions = append(conditions, "group_jid = ?")
		args = append(args, filter.GroupJID)
	}
	if filter.SenderJID != "" {
		conditions = append(conditions, "sender_jid = ?")
		args = append(args, filter.SenderJID)
	}
	if filter.Search != "" {
		conditions = append(conditions, "(incoming_message LIKE ? OR response_message LIKE ?)")
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern, pattern)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM bot_event_logs" + whereClause
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, event_type, rule_id, sender_jid, group_jid,
			   incoming_message, response_message, latency_ms, status, created_at
		FROM bot_event_logs
	` + whereClause + " ORDER BY id DESC"

	queryArgs := append([]any(nil), args...)
	if filter.Limit > 0 {
		query += " LIMIT ?"
		queryArgs = append(queryArgs, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			queryArgs = append(queryArgs, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := make([]*domainBot.EventLog, 0)
	for rows.Next() {
		item, err := r.scanEventLog(rows)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, item)
	}
	return logs, total, rows.Err()
}

func (r *SQLiteRepository) PurgeEventLogs(ctx context.Context, before *time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var res sql.Result
	var err error
	if before != nil {
		res, err = r.db.ExecContext(ctx, "DELETE FROM bot_event_logs WHERE created_at < ?", *before)
	} else {
		res, err = r.db.ExecContext(ctx, "DELETE FROM bot_event_logs")
	}
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *SQLiteRepository) DeleteLogs(ctx context.Context, before time.Time) error {
	_, err := r.PurgeEventLogs(ctx, &before)
	return err
}

func (r *SQLiteRepository) ClearLogs(ctx context.Context) error {
	_, err := r.PurgeEventLogs(ctx, nil)
	return err
}
