package api

import (
	"context"
	"fmt"

	"github.com/blacklee123/feishu-kimi/pkg/services"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type Config struct {
	FeishuAppId             string `mapstructure:"FEISHU_APP_ID"`
	FeishuAppSecret         string `mapstructure:"FEISHU_APP_SECRET"`
	FeishuEncryptKey        string `mapstructure:"FEISHU_ENCRYPT_KEY"`
	FeishuVerificationToken string `mapstructure:"FEISHU_VERIFICATION_TOKEN"`

	OpenaiApiKey    string `mapstructure:"OPENAI_KEY"`
	OpenaiModel     string `mapstructure:"OPENAI_MODEL"`
	OpenaiMaxTokens int    `mapstructure:"OPENAI_MAX_TOKENS"`
	OpenaiApiUrl    string `mapstructure:"OPENAI_API_URL"`
}

type Server struct {
	gpt          *services.ChatGPT
	logger       *zap.Logger
	config       *Config
	larkClient   *lark.Client
	larkWsClient *larkws.Client
}

func NewServer(config *Config, logger *zap.Logger) (*Server, error) {

	defaultConfig := openai.DefaultConfig(config.OpenaiApiKey)
	defaultConfig.BaseURL = config.OpenaiApiUrl
	client := openai.NewClientWithConfig(defaultConfig)
	srv := &Server{
		logger: logger,
		config: config,
		gpt: &services.ChatGPT{
			ApiKey:    config.OpenaiApiKey,
			ApiUrl:    config.OpenaiApiUrl,
			Model:     config.OpenaiModel,
			MaxTokens: config.OpenaiMaxTokens,
			Client:    client,
			Logger:    logger,
		},
		larkClient: lark.NewClient(config.FeishuAppId, config.FeishuAppSecret, lark.WithLogLevel(larkcore.LogLevelError)),
	}
	handler := NewMessageHandler(srv.gpt, *config, logger, srv.larkClient)
	eventHandler := dispatcher.NewEventDispatcher(
		config.FeishuVerificationToken, config.FeishuEncryptKey).
		OnP2MessageReceiveV1(handler.MsgReceivedHandler)
	srv.larkWsClient = larkws.NewClient(config.FeishuAppId, config.FeishuAppSecret, larkws.WithEventHandler(eventHandler), larkws.WithLogLevel(larkcore.LogLevelDebug))
	return srv, nil
}

func (s *Server) ListenAndServe() {
	// log version and port
	s.logger.Info("config: ",
		zap.String("confgi", fmt.Sprintf("%v", *s.config)),
	)
	go func() {
		err := s.larkWsClient.Start(context.Background())
		if err != nil {
			s.logger.Fatal("larkws  启动失败", zap.Error(err))
		}
	}()
}
