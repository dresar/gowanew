package bot

type TriggerType string

const (
	TriggerExact      TriggerType = "exact"
	TriggerContains   TriggerType = "contains"
	TriggerStartsWith TriggerType = "starts_with"
	TriggerRegex      TriggerType = "regex"
)

func (t TriggerType) IsValid() bool {
	switch t {
	case TriggerExact, TriggerContains, TriggerStartsWith, TriggerRegex:
		return true
	default:
		return false
	}
}

type Scope string

const (
	ScopePrivate Scope = "private"
	ScopeGroup   Scope = "group"
	ScopeAll     Scope = "all"
)

func (s Scope) IsValid() bool {
	switch s {
	case ScopePrivate, ScopeGroup, ScopeAll:
		return true
	default:
		return false
	}
}

type ResponseType string

const (
	ResponseTypeText  ResponseType = "text"
	ResponseTypeMedia ResponseType = "media"
)

func (r ResponseType) IsValid() bool {
	switch r {
	case ResponseTypeText, ResponseTypeMedia:
		return true
	default:
		return false
	}
}

type EventType string

const (
	EventTypeAutoReply       EventType = "auto_reply"
	EventTypeGroupModeration EventType = "group_moderation"
	EventTypeAIChat          EventType = "ai_chat"
	EventTypeAITool          EventType = "ai_tool"
	EventTypeError           EventType = "error"
)

func (e EventType) IsValid() bool {
	switch e {
	case EventTypeAutoReply, EventTypeGroupModeration, EventTypeAIChat, EventTypeAITool, EventTypeError:
		return true
	default:
		return false
	}
}

type LogStatus string

const (
	LogStatusSuccess     LogStatus = "success"
	LogStatusFailed      LogStatus = "failed"
	LogStatusIgnored     LogStatus = "ignored"
	LogStatusRateLimited LogStatus = "rate_limited"
)

func (s LogStatus) IsValid() bool {
	switch s {
	case LogStatusSuccess, LogStatusFailed, LogStatusIgnored, LogStatusRateLimited:
		return true
	default:
		return false
	}
}
