package core

import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

var Logger *zap.Logger

// InitLogger 初始化日志
func InitLogger(logLevel string) {
	Logger, _ = zap.NewProduction()

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		CallerKey:      "line",
		NameKey:        "hydra",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder, // 小写编码器
		EncodeTime:     TimeEncoder,                      // 自定义时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,   //
		EncodeCaller:   zapcore.ShortCallerEncoder,       // 相对路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		fmt.Println("日志级别解析失败，默认设置为info")
		level = zapcore.InfoLevel
	}

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	cores := make([]zapcore.Core, 0)
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(level)
	cores = append(cores,
		zapcore.NewCore(
			encoder, // 编码器配置
			zapcore.NewMultiWriteSyncer(getWriteSyncer()...), // 打印到控制台或文件或其他平台
			atomicLevel, // 日志级别
		),
	)

	core := zapcore.NewTee(cores...)
	// 开启开发模式，堆栈跟踪
	caller := zap.AddCaller()
	// 开启文件及行号
	development := zap.Development()
	// 设置初始化字段
	filed := zap.Fields()
	// 构造日志
	Logger = zap.New(core, caller, development, filed)
}

// TimeEncoder 序列化时间
func TimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(TimeFormat))
}

const TimeFormat = "2006-01-02 15:04:05"

func getWriteSyncer() []zapcore.WriteSyncer {
	var items []zapcore.WriteSyncer
	hook := lumberjack.Logger{
		Filename:   "chatgpt-bot.log", // 日志文件路径
		MaxSize:    100,               // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: 100,               // 日志文件最多保存多少个备份
		MaxAge:     100,               // 文件最多保存多少天
		Compress:   true,              // 是否压缩
	}
	items = append(items, zapcore.AddSync(&hook))
	items = append(items, zapcore.AddSync(os.Stdout))
	return items
}
