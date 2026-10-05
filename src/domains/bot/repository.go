package bot

import (
	"context"
	"time"
)

type IRuleRepository interface {
	CreateRule(ctx context.Context, rule *Rule) (*Rule, error)
	GetRuleByID(ctx context.Context, id int64) (*Rule, error)
	ListRules(ctx context.Context, filter RuleFilter) ([]*Rule, error)
	UpdateRule(ctx context.Context, id int64, req UpdateRuleRequest) (*Rule, error)
	DeleteRule(ctx context.Context, id int64) error
	ToggleRuleActive(ctx context.Context, id int64) (*Rule, error)
	ToggleActive(ctx context.Context, id int64) (*Rule, error)
}

type IGroupRuleRepository interface {
	GetGroupRuleByGroupJID(ctx context.Context, groupJID string) (*GroupRule, error)
	GetGroupRule(ctx context.Context, groupJID string) (*GroupRule, error)
	ListAllGroupRules(ctx context.Context) ([]*GroupRule, error)
	GetGroupRules(ctx context.Context) ([]*GroupRule, error)
	UpsertGroupRule(ctx context.Context, rule *GroupRule) (*GroupRule, error)
	DeleteGroupRule(ctx context.Context, groupJID string) error
}

type IAIConfigRepository interface {
	GetAIConfig(ctx context.Context) (*AIConfig, error)
	UpdateAIConfig(ctx context.Context, req UpdateAIConfigRequest) (*AIConfig, error)
}

type IEventLogRepository interface {
	CreateEventLog(ctx context.Context, log *EventLog) (*EventLog, error)
	CreateLog(ctx context.Context, log *EventLog) (*EventLog, error)
	ListEventLogs(ctx context.Context, filter EventLogFilter) ([]*EventLog, int64, error)
	PurgeEventLogs(ctx context.Context, before *time.Time) (int64, error)
	DeleteLogs(ctx context.Context, before time.Time) error
	ClearLogs(ctx context.Context) error
}

type IBotRepository interface {
	InitializeSchema(ctx context.Context) error
	Close() error
	IRuleRepository
	IGroupRuleRepository
	IAIConfigRepository
	IEventLogRepository
}
