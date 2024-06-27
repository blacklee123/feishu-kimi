package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/blacklee123/feishu-kimi/pkg/services"
	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"

	"go.uber.org/zap"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// 责任链
func chain(data *ActionInfo, actions ...Action) bool {
	for _, v := range actions {
		if !v.Execute(data) {
			return false
		}
	}
	return true
}

type MessageHandlerInterface interface {
	MsgReceivedHandler(ctx context.Context, event *larkim.P2MessageReceiveV1) error
}

type HandlerType string

const (
	GroupHandler = "group"
	UserHandler  = "personal"
)

func judgeChatType(event *larkim.P2MessageReceiveV1) HandlerType {
	chatType := event.Event.Message.ChatType
	if *chatType == "group" {
		return GroupHandler
	}
	if *chatType == "p2p" {
		return UserHandler
	}
	return "otherChat"
}

type MessageHandler struct {
	sessionCache services.SessionServiceCacheInterface
	gpt          *services.ChatGPT
	config       Config
	logger       *zap.Logger
	larkClient   *lark.Client
}

func judgeMsgType(event *larkim.P2MessageReceiveV1) (string, error) {
	msgType := event.Event.Message.MessageType

	switch *msgType {
	case "text", "post", "file":
		return *msgType, nil
	default:
		return "", fmt.Errorf("unknown message type: %v", *msgType)
	}

}

func (a MessageHandler) replyMsg(ctx context.Context, msg string, msgId *string) error {
	msg, i := processMessage(msg)
	if i != nil {
		return i
	}
	client := a.larkClient
	content := larkim.NewTextMsgBuilder().
		Text(msg).
		Build()

	resp, err := client.Im.Message.Reply(ctx, larkim.NewReplyMessageReqBuilder().
		MessageId(*msgId).
		Body(larkim.NewReplyMessageReqBodyBuilder().
			MsgType(larkim.MsgTypeText).
			Uuid(uuid.New().String()).
			Content(content).
			Build()).
		Build())

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Println(resp.Code, resp.Msg, resp.RequestId())
		return errors.New(resp.Msg)
	}
	return nil
}

func (m MessageHandler) MsgReceivedHandler(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	go func() {
		m.logger.Info("[receive]", zap.String("messageid", *event.Event.Message.MessageId), zap.String("MessageType", *event.Event.Message.MessageType), zap.String("message", *event.Event.Message.Content))
		// alert(ctx, fmt.Sprintf("收到消息: messageId %v", *event.Event.Message.MessageId))
		handlerType := judgeChatType(event)
		if handlerType == "otherChat" {
			m.replyMsg(ctx, "unknown chat type", event.Event.Message.MessageId)
			m.logger.Error("unknown chat type")
			return
		}
		//fmt.Println(larkcore.Prettify(event.Event.Message))

		msgType, err := judgeMsgType(event)
		if err != nil {
			m.replyMsg(ctx, "🥹不支持的消息类型, 当前仅支持文本消息、文件消息", event.Event.Message.MessageId)
			m.logger.Error("error getting message type", zap.Error(err))
			return
		}

		content := event.Event.Message.Content
		msgId := event.Event.Message.MessageId
		rootId := event.Event.Message.RootId
		chatId := event.Event.Message.ChatId

		sessionId := rootId
		if sessionId == nil || *sessionId == "" {
			sessionId = msgId
		}
		qParsed := strings.Trim(parseContent(*content, msgType), " ")
		m.logger.Info("[receive]", zap.String("messageid", *event.Event.Message.MessageId), zap.String("MessageType", *event.Event.Message.MessageType), zap.String("qParsed", qParsed))
		imageKeys := []string{}
		if msgType == "post" {
			imageKeys = parsePostImageKeys(*content)
		}

		fileKey, fileName := parseFileKey(*content)
		msgInfo := MsgInfo{
			handlerType: handlerType,
			msgType:     msgType,
			msgId:       msgId,
			chatId:      chatId,
			userId:      event.Event.Sender.SenderId.OpenId,
			qParsed:     qParsed,
			fileKey:     fileKey,
			fileName:    fileName,
			imageKeys:   imageKeys,
			sessionId:   sessionId,
		}
		data := &ActionInfo{
			ctx:        &ctx,
			handler:    &m,
			info:       &msgInfo,
			logger:     m.logger,
			config:     m.config,
			larkClient: m.larkClient,
		}
		actions := []Action{
			&HelpAction{},    //帮助处理
			&PreAction{},     //预处理
			&FileAction{},    //文件处理
			&MessageAction{}, //消息处理
		}
		chain(data, actions...)
	}()
	return nil
}

var _ MessageHandlerInterface = (*MessageHandler)(nil)

func NewMessageHandler(gpt *services.ChatGPT, config Config, logger *zap.Logger, larkClient *lark.Client) MessageHandlerInterface {
	return &MessageHandler{
		sessionCache: services.GetSessionCache(),
		gpt:          gpt,
		config:       config,
		logger:       logger,
		larkClient:   larkClient,
	}
}
