package bot

import "net/http"

type NotFoundError string

func (e NotFoundError) Error() string {
	return string(e)
}

func (e NotFoundError) ErrCode() string {
	return "NOT_FOUND"
}

func (e NotFoundError) StatusCode() int {
	return http.StatusNotFound
}

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}

func (e ValidationError) ErrCode() string {
	return "VALIDATION_ERROR"
}

func (e ValidationError) StatusCode() int {
	return http.StatusBadRequest
}

var (
	ErrRuleNotFound           = NotFoundError("bot rule not found")
	ErrGroupRuleNotFound      = NotFoundError("group rule not found")
	ErrAIConfigNotFound       = NotFoundError("ai config not found")
	ErrAIPersonaNotFound      = NotFoundError("ai persona not found")
	ErrInvalidTriggerType     = ValidationError("invalid trigger type")
	ErrInvalidScope           = ValidationError("invalid scope")
	ErrInvalidResponseType    = ValidationError("invalid response type")
	ErrInvalidEventType       = ValidationError("invalid event type")
	ErrInvalidLogStatus       = ValidationError("invalid log status")
	ErrMissingTriggerValue    = ValidationError("trigger value is required")
	ErrMissingResponseContent = ValidationError("response content is required")
	ErrMissingGroupJID        = ValidationError("group JID is required")
	ErrMissingSenderJID       = ValidationError("sender JID is required")
	ErrMissingPhoneNumber     = ValidationError("phone number is required")
	ErrMissingCustomPrompt    = ValidationError("custom prompt is required")
)
