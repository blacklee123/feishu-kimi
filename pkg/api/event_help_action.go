package api

import (
	"fmt"
	"io"
	"time"

	"github.com/blacklee123/feishu-kimi/pkg/utils"
	"go.uber.org/zap"
)

type HelpAction struct { /*帮助*/
}

func (*HelpAction) Execute(a *ActionInfo) bool {
	// if len(a.info.qParsed) == 0 {
	// 	sendMsg(*a.ctx, "🤖️：你想知道什么呢~", a.info.chatId)
	// 	fmt.Println("msgId", *a.info.msgId,
	// 		"message.text is empty")

	// 	return false
	// }
	if _, foundHelp := utils.EitherTrimEqual(a.info.qParsed, "/help", "帮助"); foundHelp {
		a.sendHelpCard(*a.ctx, a.info.sessionId, a.info.msgId)
		return false
	}
	if _, foundHelp := utils.EitherTrimEqual(a.info.qParsed, "/files"); foundHelp {
		files, err := a.handler.gpt.ListFiles(*a.ctx)
		if err != nil {
			a.logger.Error("ListFiles error", zap.Error(err))
			return false
		}
		msg := ""
		for _, file := range files.Files {
			msg += fmt.Sprintf("id: %s\n文件名: %s\n文件大小: %.2f MB\n上传时间: %s\n\n", file.ID, file.FileName, float64(file.Bytes)/1024/1024, time.Unix(file.CreatedAt, 0).Format(time.DateTime))
		}
		msg += "/read id prompt 可基于id对应的文件进行对话"
		a.replyMsg(*a.ctx, msg, a.info.msgId)
		return false
	}

	if matched, fileId := utils.MatchDeleteFile(a.info.qParsed); matched {
		err := a.handler.gpt.DeleteFile(*a.ctx, fileId)
		if err != nil {
			a.logger.Error("DeleteFile error", zap.Error(err))
			return false
		}
		a.replyMsg(*a.ctx, "删除成功", a.info.msgId)
		return false
	}

	if matched, fileId := utils.MatchRetrieveFile(a.info.qParsed); matched {
		file, err := a.handler.gpt.GetFileContent(*a.ctx, fileId)
		if err != nil {
			a.logger.Error("GetFileContent error", zap.Error(err))
			return false
		}
		msg, err := io.ReadAll(file)
		if err != nil {
			a.logger.Error("ReadAll error", zap.Error(err))
			return false
		}
		a.replyMsg(*a.ctx, string(msg), a.info.msgId)
		return false
	}
	return true
}
