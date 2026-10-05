package usecase

import (
	"context"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dresar/gowanew/config"
	domainBot "github.com/dresar/gowanew/domains/bot"
	domainChatStorage "github.com/dresar/gowanew/domains/chatstorage"
	botInfrastructure "github.com/dresar/gowanew/infrastructure/bot"
	whatsappInfrastructure "github.com/dresar/gowanew/infrastructure/whatsapp"
	pkgUtils "github.com/dresar/gowanew/pkg/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var (
	antiLinkPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)chat\.whatsapp\.com\/[a-zA-Z0-9]+`),
		regexp.MustCompile(`(?i)wa\.me\/[0-9]+`),
		regexp.MustCompile(`(?i)https?:\/\/[^\s]+`),
	}
	globalBotServiceMu sync.RWMutex
	globalBotService   *BotService
)

type ModerationResult struct {
	ShouldRevoke bool   `json:"should_revoke"`
	ShouldKick   bool   `json:"should_kick"`
	Reason       string `json:"reason"`
}

type IBotUsecase interface {
	CreateRule(ctx context.Context, req domainBot.CreateRuleRequest) (*domainBot.Rule, error)
	GetRuleByID(ctx context.Context, id int64) (*domainBot.Rule, error)
	ListRules(ctx context.Context, filter domainBot.RuleFilter) ([]*domainBot.Rule, error)
	UpdateRule(ctx context.Context, id int64, req domainBot.UpdateRuleRequest) (*domainBot.Rule, error)
	DeleteRule(ctx context.Context, id int64) error
	ToggleRuleActive(ctx context.Context, id int64) (*domainBot.Rule, error)

	GetGroupRule(ctx context.Context, groupJID string) (*domainBot.GroupRule, error)
	ListGroupRules(ctx context.Context) ([]*domainBot.GroupRule, error)
	UpsertGroupRule(ctx context.Context, req domainBot.UpsertGroupRuleRequest) (*domainBot.GroupRule, error)
	DeleteGroupRule(ctx context.Context, groupJID string) error

	GetAIConfig(ctx context.Context) (*domainBot.AIConfig, error)
	UpdateAIConfig(ctx context.Context, req domainBot.UpdateAIConfigRequest) (*domainBot.AIConfig, error)
	ChatWithAI(ctx context.Context, req domainBot.ChatRequest) (*domainBot.ChatResult, error)
	GetTools(ctx context.Context) ([]domainBot.ToolDefinition, error)
	ExecuteTool(ctx context.Context, client *whatsmeow.Client, chatStorageRepo domainChatStorage.IChatStorageRepository, req domainBot.ToolRequest) (*domainBot.ToolResult, error)

	CreateEventLog(ctx context.Context, dto domainBot.CreateEventLogDTO) (*domainBot.EventLog, error)
	ListEventLogs(ctx context.Context, filter domainBot.EventLogFilter) ([]*domainBot.EventLog, int64, error)
	ClearEventLogs(ctx context.Context) error

	MatchRule(rules []*domainBot.Rule, text string, isGroup bool) *domainBot.Rule
	ModerateMessage(ctx context.Context, groupJID string, senderJID string, messageText string) (*ModerationResult, bool)
	FormatWelcome(template string, userName string, groupName string) string
	FormatFarewell(template string, userName string, groupName string) string
	IsBotAdmin(ctx context.Context, client *whatsmeow.Client, groupJID types.JID) bool

	ExtractIncomingText(evt *events.Message) string
	ShouldIgnoreMessage(evt *events.Message, client *whatsmeow.Client) bool
	DispatchResponse(ctx context.Context, client *whatsmeow.Client, recipient types.JID, rule *domainBot.Rule) (string, error)
	HandleMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client, chatStorageRepo domainChatStorage.IChatStorageRepository) (bool, error)
	HandleGroupInfo(ctx context.Context, evt *events.GroupInfo, client *whatsmeow.Client)
}

type BotService struct {
	repo domainBot.IBotRepository
}

func NewBotService(repo domainBot.IBotRepository) *BotService {
	service := &BotService{repo: repo}
	return service
}

func SetGlobalBotService(s *BotService) {
	globalBotServiceMu.Lock()
	defer globalBotServiceMu.Unlock()
	globalBotService = s
}

func GetGlobalBotService() *BotService {
	globalBotServiceMu.RLock()
	s := globalBotService
	globalBotServiceMu.RUnlock()
	if s != nil {
		return s
	}

	globalBotServiceMu.Lock()
	defer globalBotServiceMu.Unlock()
	if globalBotService != nil {
		return globalBotService
	}

	dbPath := filepath.Join(config.PathStorages, "bot.db")
	db, err := botInfrastructure.OpenBotDB(dbPath, 1)
	if err == nil {
		repo := botInfrastructure.NewSQLiteRepository(db)
		_ = repo.InitializeSchema(context.Background())
		globalBotService = NewBotService(repo)
		return globalBotService
	}

	return nil
}

func init() {
	whatsappInfrastructure.SetBotHandler(func(ctx context.Context, evt *events.Message, chatStorageRepo domainChatStorage.IChatStorageRepository, client *whatsmeow.Client) bool {
		service := GetGlobalBotService()
		if service == nil {
			return false
		}
		handled, _ := service.HandleMessage(ctx, evt, client, chatStorageRepo)
		return handled
	})

	whatsappInfrastructure.SetBotGroupInfoHandler(func(ctx context.Context, evt *events.GroupInfo, client *whatsmeow.Client) {
		service := GetGlobalBotService()
		if service == nil {
			return
		}
		service.HandleGroupInfo(ctx, evt, client)
	})
}

func (s *BotService) CreateRule(ctx context.Context, req domainBot.CreateRuleRequest) (*domainBot.Rule, error) {
	if !req.TriggerType.IsValid() {
		return nil, domainBot.ErrInvalidTriggerType
	}
	if strings.TrimSpace(req.TriggerValue) == "" {
		return nil, domainBot.ErrMissingTriggerValue
	}
	if req.TriggerType == domainBot.TriggerRegex {
		if _, err := regexp.Compile(req.TriggerValue); err != nil {
			return nil, domainBot.ValidationError(err.Error())
		}
	}
	if !req.Scope.IsValid() {
		return nil, domainBot.ErrInvalidScope
	}
	if !req.ResponseType.IsValid() {
		return nil, domainBot.ErrInvalidResponseType
	}
	if strings.TrimSpace(req.ResponseContent) == "" {
		return nil, domainBot.ErrMissingResponseContent
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	rule := &domainBot.Rule{
		TriggerType:     req.TriggerType,
		TriggerValue:    req.TriggerValue,
		Scope:           req.Scope,
		ResponseType:    req.ResponseType,
		ResponseContent: req.ResponseContent,
		MediaURL:        req.MediaURL,
		IsActive:        isActive,
	}

	return s.repo.CreateRule(ctx, rule)
}

func (s *BotService) GetRuleByID(ctx context.Context, id int64) (*domainBot.Rule, error) {
	return s.repo.GetRuleByID(ctx, id)
}

func (s *BotService) ListRules(ctx context.Context, filter domainBot.RuleFilter) ([]*domainBot.Rule, error) {
	return s.repo.ListRules(ctx, filter)
}

func (s *BotService) UpdateRule(ctx context.Context, id int64, req domainBot.UpdateRuleRequest) (*domainBot.Rule, error) {
	if req.TriggerType != nil && !req.TriggerType.IsValid() {
		return nil, domainBot.ErrInvalidTriggerType
	}
	if req.TriggerValue != nil && strings.TrimSpace(*req.TriggerValue) == "" {
		return nil, domainBot.ErrMissingTriggerValue
	}
	if req.TriggerType != nil && *req.TriggerType == domainBot.TriggerRegex && req.TriggerValue != nil {
		if _, err := regexp.Compile(*req.TriggerValue); err != nil {
			return nil, domainBot.ValidationError(err.Error())
		}
	}
	if req.Scope != nil && !req.Scope.IsValid() {
		return nil, domainBot.ErrInvalidScope
	}
	if req.ResponseType != nil && !req.ResponseType.IsValid() {
		return nil, domainBot.ErrInvalidResponseType
	}
	if req.ResponseContent != nil && strings.TrimSpace(*req.ResponseContent) == "" {
		return nil, domainBot.ErrMissingResponseContent
	}

	return s.repo.UpdateRule(ctx, id, req)
}

func (s *BotService) DeleteRule(ctx context.Context, id int64) error {
	return s.repo.DeleteRule(ctx, id)
}

func (s *BotService) ToggleRuleActive(ctx context.Context, id int64) (*domainBot.Rule, error) {
	return s.repo.ToggleRuleActive(ctx, id)
}

func (s *BotService) GetGroupRule(ctx context.Context, groupJID string) (*domainBot.GroupRule, error) {
	return s.repo.GetGroupRuleByGroupJID(ctx, groupJID)
}

func (s *BotService) ListGroupRules(ctx context.Context) ([]*domainBot.GroupRule, error) {
	return s.repo.ListAllGroupRules(ctx)
}

func (s *BotService) UpsertGroupRule(ctx context.Context, req domainBot.UpsertGroupRuleRequest) (*domainBot.GroupRule, error) {
	if strings.TrimSpace(req.GroupJID) == "" {
		return nil, domainBot.ErrMissingGroupJID
	}

	antiLink := false
	if req.AntiLinkEnabled != nil {
		antiLink = *req.AntiLinkEnabled
	}

	welcome := false
	if req.WelcomeEnabled != nil {
		welcome = *req.WelcomeEnabled
	}

	welcomeTpl := ""
	if req.WelcomeTemplate != nil {
		welcomeTpl = *req.WelcomeTemplate
	}

	farewell := false
	if req.FarewellEnabled != nil {
		farewell = *req.FarewellEnabled
	}

	farewellTpl := ""
	if req.FarewellTemplate != nil {
		farewellTpl = *req.FarewellTemplate
	}

	rule := &domainBot.GroupRule{
		GroupJID:         req.GroupJID,
		AntiLinkEnabled:  antiLink,
		WelcomeEnabled:   welcome,
		WelcomeTemplate:  welcomeTpl,
		FarewellEnabled:  farewell,
		FarewellTemplate: farewellTpl,
	}

	return s.repo.UpsertGroupRule(ctx, rule)
}

func (s *BotService) DeleteGroupRule(ctx context.Context, groupJID string) error {
	return s.repo.DeleteGroupRule(ctx, groupJID)
}

func (s *BotService) CreateEventLog(ctx context.Context, dto domainBot.CreateEventLogDTO) (*domainBot.EventLog, error) {
	if !dto.EventType.IsValid() {
		return nil, domainBot.ErrInvalidEventType
	}
	if !dto.Status.IsValid() {
		return nil, domainBot.ErrInvalidLogStatus
	}
	if strings.TrimSpace(dto.SenderJID) == "" {
		return nil, domainBot.ErrMissingSenderJID
	}

	logEntry := &domainBot.EventLog{
		EventType:       dto.EventType,
		RuleID:          dto.RuleID,
		SenderJID:       dto.SenderJID,
		GroupJID:        dto.GroupJID,
		IncomingMessage: dto.IncomingMessage,
		ResponseMessage: dto.ResponseMessage,
		LatencyMS:       dto.LatencyMS,
		Status:          dto.Status,
	}

	return s.repo.CreateEventLog(ctx, logEntry)
}

func (s *BotService) ListEventLogs(ctx context.Context, filter domainBot.EventLogFilter) ([]*domainBot.EventLog, int64, error) {
	return s.repo.ListEventLogs(ctx, filter)
}

func (s *BotService) ClearEventLogs(ctx context.Context) error {
	return s.repo.ClearLogs(ctx)
}

func (s *BotService) ExtractIncomingText(evt *events.Message) string {
	if evt == nil || evt.Message == nil {
		return ""
	}

	msg := pkgUtils.UnwrapMessage(evt.Message)
	if msg == nil {
		return ""
	}

	if text := pkgUtils.ExtractMessageTextFromProto(msg); text != "" {
		return text
	}

	if protoMsg := msg.GetProtocolMessage(); protoMsg != nil {
		if edited := protoMsg.GetEditedMessage(); edited != nil {
			if text := pkgUtils.ExtractMessageTextFromProto(edited); text != "" {
				return text
			}
		}
	}

	return ""
}

func (s *BotService) ShouldIgnoreMessage(evt *events.Message, client *whatsmeow.Client) bool {
	if evt == nil || evt.Info.IsFromMe {
		return true
	}

	if evt.Info.IsIncomingBroadcast() {
		return true
	}

	chatStr := evt.Info.Chat.String()
	if strings.Contains(evt.Info.SourceString(), "broadcast") ||
		strings.HasSuffix(chatStr, "@broadcast") ||
		strings.HasPrefix(chatStr, "status@") ||
		evt.Info.Chat.Server == types.BroadcastServer {
		return true
	}

	if client != nil && client.Store != nil {
		if client.Store.ID != nil && evt.Info.Sender.ToNonAD() == client.Store.ID.ToNonAD() {
			return true
		}
		if !client.Store.LID.IsEmpty() && evt.Info.Sender.ToNonAD() == client.Store.LID.ToNonAD() {
			return true
		}
	}

	return false
}

func safeRegexMatch(pattern, text string) (matched bool) {
	defer func() {
		if r := recover(); r != nil {
			matched = false
		}
	}()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(text)
}

func (s *BotService) MatchRule(rules []*domainBot.Rule, text string, isGroup bool) *domainBot.Rule {
	cleanText := strings.TrimSpace(text)
	lowerText := strings.ToLower(cleanText)

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		if rule.Scope == domainBot.ScopePrivate && isGroup {
			continue
		}
		if rule.Scope == domainBot.ScopeGroup && !isGroup {
			continue
		}

		matched := false
		lowerTrigger := strings.ToLower(strings.TrimSpace(rule.TriggerValue))

		switch rule.TriggerType {
		case domainBot.TriggerExact:
			matched = strings.EqualFold(cleanText, strings.TrimSpace(rule.TriggerValue))
		case domainBot.TriggerContains:
			matched = strings.Contains(lowerText, lowerTrigger)
		case domainBot.TriggerStartsWith:
			matched = strings.HasPrefix(lowerText, lowerTrigger)
		case domainBot.TriggerRegex:
			matched = safeRegexMatch(rule.TriggerValue, cleanText)
		}

		if matched {
			return rule
		}
	}

	return nil
}

func (s *BotService) ModerateMessage(ctx context.Context, groupJID string, senderJID string, messageText string) (*ModerationResult, bool) {
	if !strings.HasSuffix(groupJID, "@g.us") {
		return nil, false
	}

	if strings.TrimSpace(messageText) == "" {
		return nil, false
	}

	groupRule, err := s.repo.GetGroupRuleByGroupJID(ctx, groupJID)
	if err != nil || groupRule == nil || !groupRule.AntiLinkEnabled {
		return nil, false
	}

	for _, pattern := range antiLinkPatterns {
		if pattern.MatchString(messageText) {
			return &ModerationResult{
				ShouldRevoke: true,
				ShouldKick:   false,
				Reason:       "anti_link_violation",
			}, true
		}
	}

	return nil, false
}

func (s *BotService) FormatWelcome(template string, userName string, groupName string) string {
	res := strings.ReplaceAll(template, "{name}", userName)
	return strings.ReplaceAll(res, "{group}", groupName)
}

func (s *BotService) FormatFarewell(template string, userName string, groupName string) string {
	res := strings.ReplaceAll(template, "{name}", userName)
	return strings.ReplaceAll(res, "{group}", groupName)
}

func (s *BotService) IsBotAdmin(ctx context.Context, client *whatsmeow.Client, groupJID types.JID) bool {
	if client == nil || client.Store == nil || client.Store.ID == nil {
		return false
	}

	groupInfo, err := client.GetGroupInfo(ctx, groupJID)
	if err != nil || groupInfo == nil {
		return false
	}

	botJID := client.Store.ID.ToNonAD()
	botLID := client.Store.LID.ToNonAD()

	for _, p := range groupInfo.Participants {
		pNonAD := p.JID.ToNonAD()
		pLIDNonAD := p.LID.ToNonAD()
		isBot := (pNonAD == botJID) || (!botLID.IsEmpty() && pLIDNonAD == botLID) || (p.JID.User == botJID.User)
		if isBot {
			return p.IsAdmin || p.IsSuperAdmin
		}
	}

	return false
}

func (s *BotService) DispatchResponse(ctx context.Context, client *whatsmeow.Client, recipient types.JID, rule *domainBot.Rule) (string, error) {
	if client == nil {
		return "MOCK-MSG-ID", nil
	}

	if rule.ResponseType == domainBot.ResponseTypeMedia && rule.MediaURL != "" {
		fileBytes, fileName, err := pkgUtils.DownloadFileFromURL(rule.MediaURL)
		if err == nil && len(fileBytes) > 0 {
			mimeType := http.DetectContentType(fileBytes)
			var waMediaType whatsmeow.MediaType
			switch {
			case strings.HasPrefix(mimeType, "image/"):
				waMediaType = whatsmeow.MediaImage
			case strings.HasPrefix(mimeType, "video/"):
				waMediaType = whatsmeow.MediaVideo
			case strings.HasPrefix(mimeType, "audio/"):
				waMediaType = whatsmeow.MediaAudio
			default:
				waMediaType = whatsmeow.MediaDocument
			}

			uploaded, err := client.Upload(ctx, fileBytes, waMediaType)
			if err == nil {
				var msg *waE2E.Message
				switch waMediaType {
				case whatsmeow.MediaImage:
					msg = &waE2E.Message{
						ImageMessage: &waE2E.ImageMessage{
							URL:           proto.String(uploaded.URL),
							DirectPath:    proto.String(uploaded.DirectPath),
							MediaKey:      uploaded.MediaKey,
							Mimetype:      proto.String(mimeType),
							FileEncSHA256: uploaded.FileEncSHA256,
							FileSHA256:    uploaded.FileSHA256,
							FileLength:    proto.Uint64(uploaded.FileLength),
							Caption:       proto.String(rule.ResponseContent),
						},
					}
				case whatsmeow.MediaVideo:
					msg = &waE2E.Message{
						VideoMessage: &waE2E.VideoMessage{
							URL:           proto.String(uploaded.URL),
							DirectPath:    proto.String(uploaded.DirectPath),
							MediaKey:      uploaded.MediaKey,
							Mimetype:      proto.String(mimeType),
							FileEncSHA256: uploaded.FileEncSHA256,
							FileSHA256:    uploaded.FileSHA256,
							FileLength:    proto.Uint64(uploaded.FileLength),
							Caption:       proto.String(rule.ResponseContent),
						},
					}
				case whatsmeow.MediaAudio:
					msg = &waE2E.Message{
						AudioMessage: &waE2E.AudioMessage{
							URL:           proto.String(uploaded.URL),
							DirectPath:    proto.String(uploaded.DirectPath),
							MediaKey:      uploaded.MediaKey,
							Mimetype:      proto.String(mimeType),
							FileEncSHA256: uploaded.FileEncSHA256,
							FileSHA256:    uploaded.FileSHA256,
							FileLength:    proto.Uint64(uploaded.FileLength),
						},
					}
				default:
					msg = &waE2E.Message{
						DocumentMessage: &waE2E.DocumentMessage{
							URL:           proto.String(uploaded.URL),
							DirectPath:    proto.String(uploaded.DirectPath),
							MediaKey:      uploaded.MediaKey,
							Mimetype:      proto.String(mimeType),
							FileName:      proto.String(fileName),
							FileEncSHA256: uploaded.FileEncSHA256,
							FileSHA256:    uploaded.FileSHA256,
							FileLength:    proto.Uint64(uploaded.FileLength),
							Caption:       proto.String(rule.ResponseContent),
						},
					}
				}

				resp, err := client.SendMessage(ctx, recipient, msg)
				if err == nil {
					return resp.ID, nil
				}
			}
		}
	}

	msg := &waE2E.Message{Conversation: proto.String(rule.ResponseContent)}
	resp, err := client.SendMessage(ctx, recipient, msg)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (s *BotService) HandleMessage(ctx context.Context, evt *events.Message, client *whatsmeow.Client, chatStorageRepo domainChatStorage.IChatStorageRepository) (bool, error) {
	start := time.Now()
	if evt == nil {
		return false, nil
	}

	text := s.ExtractIncomingText(evt)
	chatJID := evt.Info.Chat
	isGroup := strings.HasSuffix(chatJID.String(), "@g.us")

	if isGroup {
		modRes, violated := s.ModerateMessage(ctx, chatJID.String(), evt.Info.Sender.String(), text)
		if violated && modRes != nil && modRes.ShouldRevoke {
			if client != nil {
				isAdmin := s.IsBotAdmin(ctx, client, chatJID)
				if isAdmin {
					normSender := whatsappInfrastructure.NormalizeJIDFromLID(ctx, evt.Info.Sender, client)
					if normSender.Server == "lid" || normSender.Server == types.HiddenUserServer {
						normSender = types.NewJID(normSender.User, types.DefaultUserServer)
					}

					revokeMsg := client.BuildRevoke(chatJID, normSender, evt.Info.ID)
					_, sendErr := client.SendMessage(ctx, chatJID, revokeMsg)

					if modRes.ShouldKick {
						_, _ = client.UpdateGroupParticipants(ctx, chatJID, []types.JID{normSender.ToNonAD()}, whatsmeow.ParticipantChangeRemove)
					}

					status := domainBot.LogStatusSuccess
					if sendErr != nil {
						status = domainBot.LogStatusFailed
					}

					_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
						EventType:       domainBot.EventTypeGroupModeration,
						SenderJID:       normSender.String(),
						GroupJID:        chatJID.String(),
						IncomingMessage: text,
						ResponseMessage: "Message revoked due to anti-link policy",
						LatencyMS:       time.Since(start).Milliseconds(),
						Status:          status,
					})
				} else {
					_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
						EventType:       domainBot.EventTypeGroupModeration,
						SenderJID:       evt.Info.Sender.String(),
						GroupJID:        chatJID.String(),
						IncomingMessage: text,
						ResponseMessage: "Cannot revoke message: bot is not group admin",
						LatencyMS:       time.Since(start).Milliseconds(),
						Status:          domainBot.LogStatusIgnored,
					})
				}
			} else {
				_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
					EventType:       domainBot.EventTypeGroupModeration,
					SenderJID:       evt.Info.Sender.String(),
					GroupJID:        chatJID.String(),
					IncomingMessage: text,
					ResponseMessage: "Message revoked due to anti-link policy",
					LatencyMS:       time.Since(start).Milliseconds(),
					Status:          domainBot.LogStatusSuccess,
				})
			}
			return true, nil
		}
	}

	if s.ShouldIgnoreMessage(evt, client) {
		return false, nil
	}

	if strings.TrimSpace(text) == "" {
		return false, nil
	}

	activeBool := true
	rules, _ := s.repo.ListRules(ctx, domainBot.RuleFilter{IsActive: &activeBool})
	matchedRule := s.MatchRule(rules, text, isGroup)
	if matchedRule == nil {
		cfg, cfgErr := s.repo.GetAIConfig(ctx)
		if cfgErr == nil && cfg != nil {
			isAITrigger := false
			prompt := text
			if cfg.TriggerPrefix != "" && strings.HasPrefix(text, cfg.TriggerPrefix) {
				isAITrigger = true
				prompt = strings.TrimSpace(strings.TrimPrefix(text, cfg.TriggerPrefix))
			} else if cfg.AutoReplyEnabled {
				isAITrigger = true
			}
			if isAITrigger && prompt != "" {
				aiRes, aiErr := s.ChatWithAI(ctx, domainBot.ChatRequest{Message: prompt})
				if aiErr != nil {
					groupJIDStr := ""
					if isGroup {
						groupJIDStr = chatJID.String()
					}
					_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
						EventType:       domainBot.EventTypeAIChat,
						SenderJID:       evt.Info.Sender.String(),
						GroupJID:        groupJIDStr,
						IncomingMessage: text,
						ResponseMessage: "AI error: " + aiErr.Error(),
						LatencyMS:       time.Since(start).Milliseconds(),
						Status:          domainBot.LogStatusFailed,
					})
					return true, aiErr
				}
				if aiRes != nil && aiRes.Reply != "" {
					replyRule := &domainBot.Rule{
						ResponseType:    domainBot.ResponseTypeText,
						ResponseContent: aiRes.Reply,
					}
					msgID, dispatchErr := s.DispatchResponse(ctx, client, chatJID, replyRule)
					if dispatchErr == nil && chatStorageRepo != nil && client != nil {
						senderJID := whatsappInfrastructure.OwnSenderJID(client)
						_ = chatStorageRepo.StoreSentMessageWithContext(ctx, msgID, senderJID, chatJID.String(), aiRes.Reply, time.Now(), nil)
					}
					status := domainBot.LogStatusSuccess
					if dispatchErr != nil {
						status = domainBot.LogStatusFailed
					}
					groupJIDStr := ""
					if isGroup {
						groupJIDStr = chatJID.String()
					}
					_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
						EventType:       domainBot.EventTypeAIChat,
						SenderJID:       evt.Info.Sender.String(),
						GroupJID:        groupJIDStr,
						IncomingMessage: text,
						ResponseMessage: aiRes.Reply,
						LatencyMS:       time.Since(start).Milliseconds(),
						Status:          status,
					})
					return true, dispatchErr
				}
			}
		}
		return false, nil
	}

	msgID, dispatchErr := s.DispatchResponse(ctx, client, chatJID, matchedRule)
	if dispatchErr == nil && chatStorageRepo != nil && client != nil {
		senderJID := whatsappInfrastructure.OwnSenderJID(client)
		_ = chatStorageRepo.StoreSentMessageWithContext(ctx, msgID, senderJID, chatJID.String(), matchedRule.ResponseContent, time.Now(), nil)
	}

	status := domainBot.LogStatusSuccess
	if dispatchErr != nil {
		status = domainBot.LogStatusFailed
	}

	groupJIDStr := ""
	if isGroup {
		groupJIDStr = chatJID.String()
	}

	_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
		EventType:       domainBot.EventTypeAutoReply,
		RuleID:          &matchedRule.ID,
		SenderJID:       evt.Info.Sender.String(),
		GroupJID:        groupJIDStr,
		IncomingMessage: text,
		ResponseMessage: matchedRule.ResponseContent,
		LatencyMS:       time.Since(start).Milliseconds(),
		Status:          status,
	})

	return true, dispatchErr
}

func (s *BotService) HandleGroupInfo(ctx context.Context, evt *events.GroupInfo, client *whatsmeow.Client) {
	if evt == nil || client == nil {
		return
	}

	groupJID := evt.JID.String()
	groupRule, err := s.repo.GetGroupRuleByGroupJID(ctx, groupJID)
	if err != nil || groupRule == nil {
		return
	}

	groupName := evt.JID.User
	groupInfo, err := client.GetGroupInfo(ctx, evt.JID)
	if err == nil && groupInfo != nil && groupInfo.GroupName.Name != "" {
		groupName = groupInfo.GroupName.Name
	}

	if len(evt.Join) > 0 && groupRule.WelcomeEnabled && groupRule.WelcomeTemplate != "" {
		for _, participant := range evt.Join {
			if client.Store != nil {
				if client.Store.ID != nil && participant.ToNonAD() == client.Store.ID.ToNonAD() {
					continue
				}
				if !client.Store.LID.IsEmpty() && participant.ToNonAD() == client.Store.LID.ToNonAD() {
					continue
				}
			}

			start := time.Now()
			name := participant.User
			if client.Store != nil && client.Store.Contacts != nil {
				contact, err := client.Store.Contacts.GetContact(ctx, participant)
				if err == nil && contact.Found {
					if contact.FullName != "" {
						name = contact.FullName
					} else if contact.PushName != "" {
						name = contact.PushName
					} else if contact.BusinessName != "" {
						name = contact.BusinessName
					}
				}
			}

			formatted := s.FormatWelcome(groupRule.WelcomeTemplate, name, groupName)
			_, sendErr := client.SendMessage(ctx, evt.JID, &waE2E.Message{Conversation: proto.String(formatted)})

			status := domainBot.LogStatusSuccess
			if sendErr != nil {
				status = domainBot.LogStatusFailed
			}

			_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
				EventType:       domainBot.EventTypeGroupModeration,
				SenderJID:       participant.String(),
				GroupJID:        groupJID,
				IncomingMessage: "EVENT_PARTICIPANT_JOIN",
				ResponseMessage: formatted,
				LatencyMS:       time.Since(start).Milliseconds(),
				Status:          status,
			})
		}
	}

	if len(evt.Leave) > 0 && groupRule.FarewellEnabled && groupRule.FarewellTemplate != "" {
		for _, participant := range evt.Leave {
			if client.Store != nil {
				if client.Store.ID != nil && participant.ToNonAD() == client.Store.ID.ToNonAD() {
					continue
				}
				if !client.Store.LID.IsEmpty() && participant.ToNonAD() == client.Store.LID.ToNonAD() {
					continue
				}
			}

			start := time.Now()
			name := participant.User
			if client.Store != nil && client.Store.Contacts != nil {
				contact, err := client.Store.Contacts.GetContact(ctx, participant)
				if err == nil && contact.Found {
					if contact.FullName != "" {
						name = contact.FullName
					} else if contact.PushName != "" {
						name = contact.PushName
					} else if contact.BusinessName != "" {
						name = contact.BusinessName
					}
				}
			}

			formatted := s.FormatFarewell(groupRule.FarewellTemplate, name, groupName)
			_, sendErr := client.SendMessage(ctx, evt.JID, &waE2E.Message{Conversation: proto.String(formatted)})

			status := domainBot.LogStatusSuccess
			if sendErr != nil {
				status = domainBot.LogStatusFailed
			}

			_, _ = s.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
				EventType:       domainBot.EventTypeGroupModeration,
				SenderJID:       participant.String(),
				GroupJID:        groupJID,
				IncomingMessage: "EVENT_PARTICIPANT_LEAVE",
				ResponseMessage: formatted,
				LatencyMS:       time.Since(start).Milliseconds(),
				Status:          status,
			})
		}
	}
}
