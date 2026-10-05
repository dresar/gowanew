package bot

import "time"

type GroupRule struct {
	ID               int64     `json:"id" db:"id"`
	GroupJID         string    `json:"group_jid" db:"group_jid"`
	AntiLinkEnabled  bool      `json:"anti_link_enabled" db:"anti_link_enabled"`
	WelcomeEnabled   bool      `json:"welcome_enabled" db:"welcome_enabled"`
	WelcomeTemplate  string    `json:"welcome_template" db:"welcome_template"`
	FarewellEnabled  bool      `json:"farewell_enabled" db:"farewell_enabled"`
	FarewellTemplate string    `json:"farewell_template" db:"farewell_template"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type UpsertGroupRuleRequest struct {
	GroupJID         string  `json:"group_jid"`
	AntiLinkEnabled  *bool   `json:"anti_link_enabled,omitempty"`
	WelcomeEnabled   *bool   `json:"welcome_enabled,omitempty"`
	WelcomeTemplate  *string `json:"welcome_template,omitempty"`
	FarewellEnabled  *bool   `json:"farewell_enabled,omitempty"`
	FarewellTemplate *string `json:"farewell_template,omitempty"`
}
