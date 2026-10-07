package bot

import "time"

type AIConfig struct {
	ID               int64     `json:"id" db:"id"`
	Provider         string    `json:"provider" db:"provider"`
	BaseURL          string    `json:"base_url" db:"base_url"`
	APIKey           string    `json:"api_key" db:"api_key"`
	Model            string    `json:"model" db:"model"`
	SystemPrompt     string    `json:"system_prompt" db:"system_prompt"`
	Temperature      float64   `json:"temperature" db:"temperature"`
	TriggerPrefix    string    `json:"trigger_prefix" db:"trigger_prefix"`
	AutoReplyEnabled bool      `json:"auto_reply_enabled" db:"auto_reply_enabled"`
	AccessMode       string    `json:"access_mode" db:"access_mode"`
	AllowedJIDs      string    `json:"allowed_jids" db:"allowed_jids"`
	BlockedJIDs      string    `json:"blocked_jids" db:"blocked_jids"`
	AllowGroups      bool      `json:"allow_groups" db:"allow_groups"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type UpdateAIConfigRequest struct {
	Provider         *string  `json:"provider,omitempty"`
	BaseURL          *string  `json:"base_url,omitempty"`
	APIKey           *string  `json:"api_key,omitempty"`
	Model            *string  `json:"model,omitempty"`
	SystemPrompt     *string  `json:"system_prompt,omitempty"`
	Temperature      *float64 `json:"temperature,omitempty"`
	TriggerPrefix    *string  `json:"trigger_prefix,omitempty"`
	AutoReplyEnabled *bool    `json:"auto_reply_enabled,omitempty"`
	AccessMode       *string  `json:"access_mode,omitempty"`
	AllowedJIDs      *string  `json:"allowed_jids,omitempty"`
	BlockedJIDs      *string  `json:"blocked_jids,omitempty"`
	AllowGroups      *bool    `json:"allow_groups,omitempty"`
}

type AIPersona struct {
	ID               int64     `json:"id" db:"id"`
	PhoneNumber      string    `json:"phone_number" db:"phone_number"`
	ContactName      string    `json:"contact_name" db:"contact_name"`
	Relationship     string    `json:"relationship" db:"relationship"`
	CustomPrompt     string    `json:"custom_prompt" db:"custom_prompt"`
	AutoReplyEnabled bool      `json:"auto_reply_enabled" db:"auto_reply_enabled"`
	UseMemory        bool      `json:"use_memory" db:"use_memory"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type CreateAIPersonaRequest struct {
	PhoneNumber      string `json:"phone_number"`
	ContactName      string `json:"contact_name,omitempty"`
	Relationship     string `json:"relationship,omitempty"`
	CustomPrompt     string `json:"custom_prompt"`
	AutoReplyEnabled *bool  `json:"auto_reply_enabled,omitempty"`
	UseMemory        *bool  `json:"use_memory,omitempty"`
	IsActive         *bool  `json:"is_active,omitempty"`
}

type UpdateAIPersonaRequest struct {
	PhoneNumber      *string `json:"phone_number,omitempty"`
	ContactName      *string `json:"contact_name,omitempty"`
	Relationship     *string `json:"relationship,omitempty"`
	CustomPrompt     *string `json:"custom_prompt,omitempty"`
	AutoReplyEnabled *bool   `json:"auto_reply_enabled,omitempty"`
	UseMemory        *bool   `json:"use_memory,omitempty"`
	IsActive         *bool   `json:"is_active,omitempty"`
}
