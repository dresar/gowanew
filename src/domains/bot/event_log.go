package bot

import "time"

type EventLog struct {
	ID              int64     `json:"id" db:"id"`
	EventType       EventType `json:"event_type" db:"event_type"`
	RuleID          *int64    `json:"rule_id,omitempty" db:"rule_id"`
	SenderJID       string    `json:"sender_jid" db:"sender_jid"`
	GroupJID        string    `json:"group_jid" db:"group_jid"`
	IncomingMessage string    `json:"incoming_message" db:"incoming_message"`
	ResponseMessage string    `json:"response_message" db:"response_message"`
	LatencyMS       int64     `json:"latency_ms" db:"latency_ms"`
	Status          LogStatus `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type EventLogFilter struct {
	EventType *EventType `json:"event_type,omitempty"`
	Status    *LogStatus `json:"status,omitempty"`
	GroupJID  string     `json:"group_jid,omitempty"`
	SenderJID string     `json:"sender_jid,omitempty"`
	Search    string     `json:"search,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

type CreateEventLogDTO struct {
	EventType       EventType `json:"event_type"`
	RuleID          *int64    `json:"rule_id,omitempty"`
	SenderJID       string    `json:"sender_jid"`
	GroupJID        string    `json:"group_jid,omitempty"`
	IncomingMessage string    `json:"incoming_message,omitempty"`
	ResponseMessage string    `json:"response_message,omitempty"`
	LatencyMS       int64     `json:"latency_ms"`
	Status          LogStatus `json:"status"`
}
