package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/blacklee123/feishu-kimi/pkg/api"
	"github.com/blacklee123/feishu-kimi/pkg/version"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	fs := pflag.NewFlagSet("default", pflag.ContinueOnError)
	fs.String("FEISHU_APP_ID", "", "FEISHU_APP_ID")
	fs.String("FEISHU_APP_SECRET", "", "FEISHU_APP_SECRET")
	fs.String("FEISHU_ENCRYPT_KEY", "", "FEISHU_ENCRYPT_KEY")
	fs.String("FEISHU_VERIFICATION_TOKEN", "", "FEISHU_VERIFICATION_TOKEN")
	fs.String("OPENAI_MODEL", "", "OPENAI_MODEL")
	fs.String("OPENAI_KEY", "", "OPENAI_KEY")
	fs.Int("OPENAI_MAX_TOKENS", 2000, "OPENAI_MAX_TOKENS")
	fs.String("OPENAI_API_URL", "https://api.openai.com/v1", "OPENAI_API_URL")

	fs.String("config-path", "config", "config dir path")
	fs.String("config", "config.yaml", "apiserver config file path")
	fs.String("level", "info", "log level debug, info, warn, error, fatal or panic")

	versionFlag := fs.BoolP("version", "v", false, "get version number")

	err := fs.Parse(os.Args[1:])
	switch {
	case err == pflag.ErrHelp:
		os.Exit(0)
	case err != nil:
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err.Error())
		fs.PrintDefaults()
		os.Exit(2)
	case *versionFlag:
		fmt.Println(version.VERSION)
		os.Exit(0)
	}
	viper.BindPFlags(fs)
	viper.AutomaticEnv()

	// load config from file
	if _, fileErr := os.Stat(filepath.Join(viper.GetString("config-path"), viper.GetString("config"))); fileErr == nil {
		viper.SetConfigName(strings.Split(viper.GetString("config"), ".")[0])
		viper.AddConfigPath(viper.GetString("config-path"))
		if readErr := viper.ReadInConfig(); readErr != nil {
			fmt.Printf("Error reading config file, %v\n", readErr)
		}
	}

	// configure logging
	logger, _ := initZap(viper.GetString("level"))
	defer logger.Sync()
	stdLog := zap.RedirectStdLog(logger)
	defer stdLog()

	var config api.Config
	// 打印当前从viper读取到的所有配置
	log.Printf("Current Viper settings: %v\n", viper.AllSettings())
	if err := viper.Unmarshal(&config); err != nil {
		log.Panic("config unmarshal failed", err)
	}
	// 绑定设置到config结构体并确保值都成功加载
	log.Printf("Unmarshaled configuration: %+v\n", config)
	srv, err := api.NewServer(&config, logger)
	if err != nil {
		logger.Fatal("初始化失败", zap.Error(err))
	}
	srv.ListenAndServe()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

}

func initZap(logLevel string) (*zap.Logger, error) {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	switch logLevel {
	case "debug":
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "fatal":
		level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	case "panic":
		level = zap.NewAtomicLevelAt(zapcore.PanicLevel)
	}

	zapEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	zapConfig := zap.Config{
		Level:       level,
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zapEncoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return zapConfig.Build()
}
