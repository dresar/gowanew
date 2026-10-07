package mcp

import (
	"github.com/dresar/gowanew/config"
	domainApp "github.com/dresar/gowanew/domains/app"
	domainChat "github.com/dresar/gowanew/domains/chat"
	domainDevice "github.com/dresar/gowanew/domains/device"
	domainGroup "github.com/dresar/gowanew/domains/group"
	domainMessage "github.com/dresar/gowanew/domains/message"
	domainSend "github.com/dresar/gowanew/domains/send"
	domainUser "github.com/dresar/gowanew/domains/user"
	"github.com/dresar/gowanew/usecase"
	"github.com/mark3labs/mcp-go/server"
)

type Deps struct {
	App      domainApp.IAppUsecase
	Send     domainSend.ISendUsecase
	Schedule domainSend.IScheduleUsecase
	Chat     domainChat.IChatUsecase
	User     domainUser.IUserUsecase
	Message  domainMessage.IMessageUsecase
	Group    domainGroup.IGroupUsecase
	Bot      usecase.IBotUsecase
	Device   domainDevice.IDeviceUsecase
}

func NewServer(deps Deps, resolver deviceResolver) *server.MCPServer {
	s := server.NewMCPServer(
		"WhatsApp Web Multidevice MCP Server",
		config.AppVersion,
		server.WithToolCapabilities(true),
		server.WithInputSchemaValidation(),
	)
	InitMcpSend(deps.Send, resolver).AddSendTools(s)
	InitMcpSchedule(deps.Schedule, resolver).AddScheduleTools(s)
	InitMcpMessage(deps.Message, resolver).AddMessageTools(s)
	InitMcpChat(deps.Chat, deps.User, resolver).AddChatTools(s)
	InitMcpGroup(deps.Group, resolver).AddGroupTools(s)
	InitMcpApp(deps.App, resolver).AddAppTools(s)
	InitMcpBot(deps.Bot, resolver).AddBotTools(s)
	InitMcpWebhook(deps.Device, resolver).AddWebhookTools(s)
	return s
}
