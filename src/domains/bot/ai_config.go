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
}
