package bot

import (
	"context"
	"database/sql"
	"fmt"
)

type schemaMigration struct {
	version    int
	statements []string
}

var migrations = []schemaMigration{
	{
		version: 1,
		statements: []string{
			`CREATE TABLE IF NOT EXISTS bot_rules (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				trigger_type TEXT NOT NULL CHECK (trigger_type IN ('exact', 'contains', 'starts_with', 'regex')),
				trigger_value TEXT NOT NULL,
				scope TEXT NOT NULL CHECK (scope IN ('private', 'group', 'all')),
				response_type TEXT NOT NULL CHECK (response_type IN ('text', 'media')),
				response_content TEXT NOT NULL,
				media_url TEXT NOT NULL DEFAULT '',
				is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
	},
	{
		version: 2,
		statements: []string{
			`CREATE INDEX IF NOT EXISTS idx_bot_rules_is_active ON bot_rules(is_active)`,
			`CREATE INDEX IF NOT EXISTS idx_bot_rules_scope ON bot_rules(scope)`,
			`CREATE INDEX IF NOT EXISTS idx_bot_rules_trigger_type ON bot_rules(trigger_type)`,
		},
	},
	{
		version: 3,
		statements: []string{
			`CREATE TABLE IF NOT EXISTS bot_group_rules (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				group_jid TEXT NOT NULL UNIQUE,
				anti_link_enabled INTEGER NOT NULL DEFAULT 0 CHECK (anti_link_enabled IN (0, 1)),
				welcome_enabled INTEGER NOT NULL DEFAULT 0 CHECK (welcome_enabled IN (0, 1)),
				welcome_template TEXT NOT NULL DEFAULT '',
				farewell_enabled INTEGER NOT NULL DEFAULT 0 CHECK (farewell_enabled IN (0, 1)),
				farewell_template TEXT NOT NULL DEFAULT '',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
	},
	{
		version: 4,
		statements: []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_bot_group_rules_group_jid ON bot_group_rules(group_jid)`,
		},
	},
	{
		version: 5,
		statements: []string{
			`CREATE TABLE IF NOT EXISTS bot_ai_config (
				id INTEGER PRIMARY KEY CHECK (id = 1),
				provider TEXT NOT NULL DEFAULT 'openai',
				base_url TEXT NOT NULL DEFAULT 'https://api.openai.com/v1',
				api_key TEXT NOT NULL DEFAULT '',
				model TEXT NOT NULL DEFAULT 'gpt-4o-mini',
				system_prompt TEXT NOT NULL DEFAULT '',
				temperature REAL NOT NULL DEFAULT 0.7,
				trigger_prefix TEXT NOT NULL DEFAULT '!ai',
				auto_reply_enabled INTEGER NOT NULL DEFAULT 0 CHECK (auto_reply_enabled IN (0, 1)),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
	},
	{
		version: 6,
		statements: []string{
			`INSERT OR IGNORE INTO bot_ai_config (
				id, provider, base_url, api_key, model,
				system_prompt, temperature, trigger_prefix,
				auto_reply_enabled, created_at, updated_at
			) VALUES (
				1, 'openai', 'https://api.openai.com/v1', '', 'gpt-4o-mini',
				'', 0.7, '!ai',
				0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
			)`,
		},
	},
	{
		version: 7,
		statements: []string{
			`CREATE TABLE IF NOT EXISTS bot_event_logs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				event_type TEXT NOT NULL CHECK (event_type IN ('auto_reply', 'group_moderation', 'ai_chat', 'ai_tool', 'error')),
				rule_id INTEGER,
				sender_jid TEXT NOT NULL,
				group_jid TEXT NOT NULL DEFAULT '',
				incoming_message TEXT NOT NULL DEFAULT '',
				response_message TEXT NOT NULL DEFAULT '',
				latency_ms INTEGER NOT NULL DEFAULT 0,
				status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'ignored', 'rate_limited')),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)`,
		},
	},
	{
		version: 8,
		statements: []string{
			`CREATE INDEX IF NOT EXISTS idx_bot_event_logs_created_at ON bot_event_logs(created_at DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_bot_event_logs_event_type ON bot_event_logs(event_type)`,
			`CREATE INDEX IF NOT EXISTS idx_bot_event_logs_sender_jid ON bot_event_logs(sender_jid)`,
			`CREATE INDEX IF NOT EXISTS idx_bot_event_logs_group_jid ON bot_event_logs(group_jid)`,
		},
	},
	{
		version: 9,
		statements: []string{
			`ALTER TABLE bot_rules ADD COLUMN recipient_jid TEXT NOT NULL DEFAULT ''`,
			`CREATE INDEX IF NOT EXISTS idx_bot_rules_recipient_jid ON bot_rules(recipient_jid)`,
		},
	},
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	bootstrapSQL := `CREATE TABLE IF NOT EXISTS bot_schema_info (
		version INTEGER PRIMARY KEY,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.ExecContext(ctx, bootstrapSQL); err != nil {
		return fmt.Errorf("bootstrap bot_schema_info: %w", err)
	}

	appliedVersions := make(map[int]bool)
	rows, err := db.QueryContext(ctx, "SELECT version FROM bot_schema_info")
	if err != nil {
		return fmt.Errorf("query bot_schema_info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return fmt.Errorf("scan migration version: %w", err)
		}
		appliedVersions[v] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate migration versions: %w", err)
	}

	for _, m := range migrations {
		if appliedVersions[m.version] {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", m.version, err)
		}

		for _, stmt := range m.statements {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("exec migration %d statement: %w", m.version, err)
			}
		}

		recordSQL := "INSERT INTO bot_schema_info (version) VALUES (?)"
		if _, err := tx.ExecContext(ctx, recordSQL, m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
	}

	return nil
}
