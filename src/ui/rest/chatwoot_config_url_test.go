package rest

import (
	"testing"

	"github.com/dresar/gowanew/config"
)

func TestPerDeviceWebhookURLPreservesQuery(t *testing.T) {
	originalWebhookURL := config.ChatwootWebhookURL
	originalBasePath := config.AppBasePath
	defer func() {
		config.ChatwootWebhookURL = originalWebhookURL
		config.AppBasePath = originalBasePath
	}()

	config.ChatwootWebhookURL = "https://gowa.testdomain.com/chatwoot/webhook?secret=the-web-secret"
	config.AppBasePath = ""

	const want = "https://gowa.testdomain.com/chatwoot/webhook/the-device-id?secret=the-web-secret"
	if got := perDeviceWebhookURL("the-device-id"); got != want {
		t.Fatalf("perDeviceWebhookURL() = %q, want %q", got, want)
	}
}
