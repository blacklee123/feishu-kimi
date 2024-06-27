package api

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

type FileAction struct { /*文件*/
}

func (*FileAction) Execute(a *ActionInfo) bool {
	if a.info.msgType == "file" {
		err := a.downloadFile(a.info.fileKey, a.info.fileName, a.info.msgId)
		if err != nil {
			fmt.Println(err)
			return false
		}
		file, err := a.handler.gpt.CreateFile(*a.ctx, a.info.fileName)
		if err != nil {
			a.sendMsg(*a.ctx, fmt.Sprintf("🤖️：文件上传失败\n错误信息: %v", err), a.info.msgId)
			return false
		}
		// 将时间戳转换为time.Time类型
		msg := fmt.Sprintf("🤖️：文件上传成功\nid: %s\n文件名: %s\n文件大小: %.2f MB\n 上传时间: %s\n\n", file.ID, file.FileName, float64(file.Bytes)/1024/1024, time.Unix(file.CreatedAt, 0).Format(time.DateTime))
		msg += fmt.Sprintf("/read %s prompt 可基于本文件进行对话", file.ID)
		err = a.updateFinalCard(*a.ctx, msg, a.info.cardId, false)
		if err != nil {
			a.logger.Error("updateFinalCard error", zap.Error(err))
			return false
		}
		defer os.Remove(a.info.fileName)
		return false
	}
	return true
}
