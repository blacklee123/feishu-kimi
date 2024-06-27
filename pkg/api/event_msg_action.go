package api

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/blacklee123/feishu-kimi/pkg/utils"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type MessageAction struct { /*消息*/
}

func (*MessageAction) Execute(a *ActionInfo) bool {
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	if a.info.newTopic {
		userName := ""
		user, err := a.retrieveUserInfo(*a.ctx, *a.info.userId)
		if err != nil {
			userName = *a.info.userId
		} else {
			userName = *user.Name
		}
		msg = append(msg, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleSystem,
			Content: fmt.Sprintf(`你是 Kimi，由 Moonshot AI 提供的人工智能助手，你更擅长中文和英文的对话。你会为用户提供安全，有帮助，准确的回答。同时，你会拒绝一切涉及恐怖主义，种族歧视，黄色暴力等问题的回答。Moonshot AI 为专有名词，不可翻译成其他语言。
			我的名字是%s, 请使用这个名字和我交流。`, userName),
			Name: "Kimi",
		})
		if matched, fileId, prompt := utils.MatchReadFile(a.info.qParsed); matched {
			file, err := a.handler.gpt.GetFileContent(*a.ctx, fileId)
			if err != nil {
				a.logger.Error("GetFileContent error", zap.Error(err))
				return false
			}
			fileContent, err := io.ReadAll(file)
			if err != nil {
				a.logger.Error("ReadAll error", zap.Error(err))
				return false
			}
			msg = append(msg, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: string(fileContent),
				Name:    "Kimi",
			})
			a.info.qParsed = prompt
		}

	}
	msg = append(msg, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: a.info.qParsed,
		Name:    *a.info.userId,
	})
	answer := ""
	chatResponseStream := make(chan string)
	go func() {
		if err := a.handler.gpt.StreamChat(*a.ctx, msg, chatResponseStream); err != nil {
			a.logger.Error("StreamChat error", zap.Error(err))
			err := a.updateFinalCard(*a.ctx, "聊天失败", a.info.cardId, a.info.newTopic)
			if err != nil {
				a.logger.Error("updateFinalCard error", zap.Error(err))
				return
			}
		}
	}()
	timer := time.NewTicker(700 * time.Millisecond)
	for {
		select {
		case <-timer.C:
			a.logger.Debug("answer", zap.String("answer", answer))
			if answer != "" {
				err := a.UpdateTextCard(*a.ctx, answer, a.info.cardId, a.info.newTopic)
				if err != nil {
					a.logger.Error("UpdateTextCard error", zap.Error(err))
				}
			}

		case res, ok := <-chatResponseStream:
			if ok {
				answer += res
			} else {
				timer.Stop()
				err := a.updateFinalCard(*a.ctx, answer, a.info.cardId, a.info.newTopic)
				if err != nil {
					a.logger.Error("updateFinalCard error", zap.Error(err))
					return false
				}
				msg := append(msg, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: answer,
					Name:    msg[0].Name,
				})
				a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
				return false
			}

		}
	}

}

func (a *ActionInfo) replyWithErrorMsg(ctx context.Context, err error, msgId *string) {
	a.replyMsg(ctx, fmt.Sprintf("🤖️：图片下载失败，请稍后再试～\n 错误信息: %v", err), msgId)
}

func createMultipleVisionMessages(query string, base64Images []string, userId string) openai.ChatCompletionMessage {
	content := []openai.ChatMessagePart{{Type: "text", Text: query}}
	for _, base64Image := range base64Images {
		content = append(content, openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: "data:image/jpeg;base64," + base64Image,
			},
		})
	}
	return openai.ChatCompletionMessage{
		Role:         openai.ChatMessageRoleUser,
		MultiContent: content,
		Name:         userId,
	}
}
