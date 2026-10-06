package bot

import "time"

type Rule struct {
	ID              int64        `json:"id" db:"id"`
	TriggerType     TriggerType  `json:"trigger_type" db:"trigger_type"`
	TriggerValue    string       `json:"trigger_value" db:"trigger_value"`
	RecipientJID    string       `json:"recipient_jid" db:"recipient_jid"`
	Scope           Scope        `json:"scope" db:"scope"`
	ResponseType    ResponseType `json:"response_type" db:"response_type"`
	ResponseContent string       `json:"response_content" db:"response_content"`
	MediaURL        string       `json:"media_url" db:"media_url"`
	IsActive        bool         `json:"is_active" db:"is_active"`
	CreatedAt       time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at" db:"updated_at"`
}

type RuleFilter struct {
	IsActive     *bool   `json:"is_active,omitempty"`
	Scope        *Scope  `json:"scope,omitempty"`
	RecipientJID *string `json:"recipient_jid,omitempty"`
	Search       string  `json:"search,omitempty"`
	Limit        int     `json:"limit,omitempty"`
	Offset       int     `json:"offset,omitempty"`
}

type CreateRuleRequest struct {
	TriggerType     TriggerType  `json:"trigger_type"`
	TriggerValue    string       `json:"trigger_value"`
	RecipientJID    string       `json:"recipient_jid,omitempty"`
	Scope           Scope        `json:"scope"`
	ResponseType    ResponseType `json:"response_type"`
	ResponseContent string       `json:"response_content"`
	MediaURL        string       `json:"media_url,omitempty"`
	IsActive        *bool        `json:"is_active,omitempty"`
}

type UpdateRuleRequest struct {
	TriggerType     *TriggerType  `json:"trigger_type,omitempty"`
	TriggerValue    *string       `json:"trigger_value,omitempty"`
	RecipientJID    *string       `json:"recipient_jid,omitempty"`
	Scope           *Scope        `json:"scope,omitempty"`
	ResponseType    *ResponseType `json:"response_type,omitempty"`
	ResponseContent *string       `json:"response_content,omitempty"`
	MediaURL        *string       `json:"media_url,omitempty"`
	IsActive        *bool         `json:"is_active,omitempty"`
}
