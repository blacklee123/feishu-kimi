package services

import (
	"context"
	"errors"
	"io"

	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type ChatGPT struct {
	ApiKey    string
	ApiUrl    string
	Model     string
	MaxTokens int
	Client    *openai.Client
	Logger    *zap.Logger
}

func (gpt *ChatGPT) Completions(ctx context.Context, msg []openai.ChatCompletionMessage) (openai.ChatCompletionMessage, error) {
	resp, err := gpt.Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:            gpt.Model,
		Messages:         msg,
		MaxTokens:        gpt.MaxTokens,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	},
	)

	if err != nil {
		gpt.Logger.Error("ChatCompletion error", zap.Error(err))
		return openai.ChatCompletionMessage{}, err
	}

	return resp.Choices[0].Message, nil
}

func (gpt *ChatGPT) StreamChat(ctx context.Context, msgs []openai.ChatCompletionMessage, responseStream chan<- string) error {
	defer close(responseStream)
	req := openai.ChatCompletionRequest{
		Model:     gpt.Model,
		Messages:  msgs,
		MaxTokens: 2000,
		Stream:    true,
	}
	stream, err := gpt.Client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		gpt.Logger.Error("ChatCompletionStream error", zap.Error(err))
		return err
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			gpt.Logger.Error("Stream error", zap.Error(err))
			return err
		}
		if len(response.Choices) > 0 {
			responseStream <- response.Choices[0].Delta.Content
			gpt.Logger.Debug("response", zap.String("content", response.Choices[0].Delta.Content))
		}

	}
}

func (gpt *ChatGPT) CreateFile(ctx context.Context, filePath string) (*openai.File, error) {
	file, err := gpt.Client.CreateFile(ctx, openai.FileRequest{
		FilePath: filePath,
		Purpose:  "file-extract",
	},
	)

	if err != nil {
		gpt.Logger.Error("CreateFile error", zap.Error(err))
		return nil, err
	}

	return &file, nil
}

func (gpt *ChatGPT) ListFiles(ctx context.Context) (*openai.FilesList, error) {
	files, err := gpt.Client.ListFiles(ctx)

	if err != nil {
		gpt.Logger.Error("CreateFile error", zap.Error(err))
		return nil, err
	}

	return &files, nil
}

func (gpt *ChatGPT) DeleteFile(ctx context.Context, fileID string) error {
	err := gpt.Client.DeleteFile(ctx, fileID)

	if err != nil {
		gpt.Logger.Error("DeleteFile error", zap.Error(err))
		return err
	}

	return nil
}

func (gpt *ChatGPT) GetFile(ctx context.Context, fileID string) (*openai.File, error) {
	file, err := gpt.Client.GetFile(ctx, fileID)

	if err != nil {
		gpt.Logger.Error("GetFile error", zap.Error(err))
		return nil, err
	}

	return &file, nil
}

func (gpt *ChatGPT) GetFileContent(ctx context.Context, fileID string) (*openai.RawResponse, error) {
	file, err := gpt.Client.GetFileContent(ctx, fileID)

	if err != nil {
		gpt.Logger.Error("GetFileContent error", zap.Error(err))
		return nil, err
	}

	return &file, nil
}
