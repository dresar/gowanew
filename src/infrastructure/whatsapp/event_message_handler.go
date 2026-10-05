package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dresar/gowanew/config"
	domainChatStorage "github.com/dresar/gowanew/domains/chatstorage"
	"github.com/dresar/gowanew/pkg/utils"
	"github.com/sirupsen/logrus"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	botHandlerMu sync.RWMutex
	botHandler   func(ctx context.Context, evt *events.Message, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) bool
)

func SetBotHandler(handler func(ctx context.Context, evt *events.Message, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) bool) {
	botHandlerMu.Lock()
	defer botHandlerMu.Unlock()
	botHandler = handler
}

func GetBotHandler() func(ctx context.Context, evt *events.Message, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) bool {
	botHandlerMu.RLock()
	defer botHandlerMu.RUnlock()
	return botHandler
}

func handleMessage(ctx context.Context, evt *events.Message, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) {
	metaParts := buildMessageMetaParts(evt)
	log.Infof("Received message %s from %s (%s): %+v",
		evt.Info.ID,
		evt.Info.SourceString(),
		strings.Join(metaParts, ", "),
		evt.Message,
	)

	evt = materializeSecretEditMessage(ctx, evt, client)
	pollPayload := preparePollWebhookPayload(ctx, client, chatStorageRepo, evt)

	if isReactionMessage(evt) {
		if err := chatStorageRepo.CreateReaction(ctx, evt); err != nil {
			log.Errorf("Failed to store incoming reaction %s: %v", evt.Info.ID, err)
		}

		handleWebhookForward(ctx, evt, client, pollPayload)
		return
	}

	if err := chatStorageRepo.CreateMessage(ctx, evt); err != nil {
		log.Errorf("Failed to store incoming message %s: %v", evt.Info.ID, err)
	}

	handleImageMessage(ctx, evt, client)

	handleAutoMarkRead(ctx, evt, client)

	handled := false
	botHandlerMu.RLock()
	bh := botHandler
	botHandlerMu.RUnlock()
	if bh != nil {
		handled = bh(ctx, evt, chatStorageRepo, client)
	}

	if !handled {
		handleAutoReply(ctx, evt, chatStorageRepo, client)
	}

	handleWebhookForward(ctx, evt, client, pollPayload)
}

func buildMessageMetaParts(evt *events.Message) []string {
	metaParts := []string{
		fmt.Sprintf("pushname: %s", evt.Info.PushName),
		fmt.Sprintf("timestamp: %s", evt.Info.Timestamp),
	}
	if evt.Info.Type != "" {
		metaParts = append(metaParts, fmt.Sprintf("type: %s", evt.Info.Type))
	}
	if evt.Info.Category != "" {
		metaParts = append(metaParts, fmt.Sprintf("category: %s", evt.Info.Category))
	}
	if evt.IsViewOnce {
		metaParts = append(metaParts, "view once")
	}
	return metaParts
}

func shouldIgnoreImageDownload(autoDownloadMedia, ignoreStatusMedia bool, chatJID types.JID) bool {
	if !autoDownloadMedia {
		return true
	}
	if ignoreStatusMedia && chatJID.Server == types.BroadcastServer && !chatJID.IsBroadcastList() {
		return true
	}
	return false
}

func handleImageMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	if shouldIgnoreImageDownload(config.WhatsappAutoDownloadMedia, config.WhatsappIgnoreStatusMedia, evt.Info.Chat) {
		return
	}
	if client == nil {
		return
	}
	if img := evt.Message.GetImageMessage(); img != nil {
		if extracted, err := utils.ExtractMedia(ctx, client, config.PathStorages, img); err != nil {
			log.Errorf("Failed to download image: %v", err)
		} else {
			log.Infof("Image downloaded to %s", extracted.MediaPath)
		}
	}
}

func handleAutoMarkRead(ctx context.Context, evt *events.Message, client *whatsmeow.Client) {
	if !config.WhatsappAutoMarkRead || evt.Info.IsFromMe {
		return
	}

	if client == nil {
		return
	}

	messageIDs := []types.MessageID{evt.Info.ID}
	timestamp := time.Now()
	chat := evt.Info.Chat
	sender := evt.Info.Sender

	if err := client.MarkRead(ctx, messageIDs, timestamp, chat, sender); err != nil {
		log.Warnf("Failed to mark message %s as read: %v", evt.Info.ID, err)
	} else {
		log.Debugf("Marked message %s as read", evt.Info.ID)
	}
}

func materializeSecretEditMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client) *events.Message {
	if evt == nil || evt.Message == nil || client == nil {
		return evt
	}
	msg := utils.UnwrapMessage(evt.Message)
	sem := msg.GetSecretEncryptedMessage()
	if sem == nil || sem.GetSecretEncType() != waE2E.SecretEncryptedMessage_MESSAGE_EDIT {
		return evt
	}
	decrypted, err := client.DecryptSecretEncryptedMessage(ctx, evt)
	if err != nil {
		targetID := ""
		if k := sem.GetTargetMessageKey(); k != nil {
			targetID = k.GetID()
		}
		log.Warnf("Failed to decrypt SecretEncryptedMessage(MESSAGE_EDIT) for %s (target=%s): %v", evt.Info.ID, targetID, err)
		return evt
	}
	if decrypted == nil {
		return evt
	}
	cloned := *evt
	cloned.Message = decrypted
	return &cloned
}

func handleWebhookForward(ctx context.Context, evt *events.Message, client *whatsmeow.Client, preparedPoll ...*webhookPollPayload) {
	if protocolMessage := evt.Message.GetProtocolMessage(); protocolMessage != nil {
		protocolType := protocolMessage.GetType().String()
		switch protocolType {
		case "REVOKE", "MESSAGE_EDIT":
		default:
			log.Debugf("Skipping webhook for protocol message type: %s", protocolType)
			return
		}
	}

	if strings.Contains(evt.Info.SourceString(), "broadcast") {
		return
	}

	var pollPayload *webhookPollPayload
	if len(preparedPoll) > 0 {
		pollPayload = preparedPoll[0]
	}
	go func(e *events.Message, c *whatsmeow.Client, poll *webhookPollPayload) {
		webhookCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if err := forwardMessageToWebhook(webhookCtx, c, e, poll); err != nil {
			logrus.Error("Failed forward to webhook: ", err)
		}
	}(evt, client, pollPayload)
}
