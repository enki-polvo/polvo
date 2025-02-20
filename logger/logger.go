package logger

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type PolvoLogger interface {
	Logger() *zap.Logger
	LogStream() *lumberjack.Logger
	Close()
	PrintInfo(fmtStr string, args ...any)
	PrintError(fmtStr string, args ...any)
}

type polvoLogger struct {
	logger  *zap.Logger
	logfile *lumberjack.Logger
	logPath string
}

func NewLogger(logPath string) PolvoLogger {
	new := new(polvoLogger)

	new.logPath = logPath
	//set log writer
	logWriter := new.initLogWriter()
	//set encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// encoderConfig := zapcore.EncoderConfig{
	// 	TimeKey:        "timestamp",
	// 	LevelKey:       "level",
	// 	NameKey:        "logger",
	// 	CallerKey:      "caller",
	// 	MessageKey:     "msg",
	// 	StacktraceKey:  "stacktrace",
	// 	LineEnding:     zapcore.DefaultLineEnding,
	// 	EncodeLevel:    zapcore.CapitalLevelEncoder,   // 대문자 레벨 (INFO, ERROR)
	// 	EncodeTime:     zapcore.ISO8601TimeEncoder,    // ISO8601 시간 형식
	// 	EncodeDuration: zapcore.StringDurationEncoder, // 지속 시간 형식
	// 	EncodeCaller:   zapcore.ShortCallerEncoder,    // 호출자 짧은 형식
	// }
	//set log core
	logCore := new.initLogCore(encoderConfig, logWriter)
	//set logger
	new.logger = zap.New(logCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	fmt.Fprintf(os.Stderr, "Log file path: %s\n", new.logfile.Filename)
	return new
}

func (logg *polvoLogger) Logger() *zap.Logger {
	return logg.logger
}

func (logg *polvoLogger) LogStream() *lumberjack.Logger {
	return logg.logfile
}

func (logg *polvoLogger) initLogWriter() zapcore.WriteSyncer {
	logg.logfile = &lumberjack.Logger{
		Filename:   logg.getLogFileName(), // log file path
		MaxSize:    128,                   // max file size：M
		MaxBackups: 30,                    // maximum number of backups
		MaxAge:     30,                    // log file save days
		Compress:   true,                  // compressed
	}
	return zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(logg.logfile), // lumberjack
		// zapcore.AddSync(os.Stderr),    // 표준 출력
	)
}

func (logg *polvoLogger) initLogCore(config zapcore.EncoderConfig, logWritter zapcore.WriteSyncer) zapcore.Core {
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(config),
			logWritter,
			zapcore.DebugLevel,
		),
	)
	return core
}

func (logg *polvoLogger) Close() {
	logg.logger.Sync()
	logg.logfile.Close()
}

func (logg *polvoLogger) PrintInfo(fmtStr string, args ...any) {
	logg.logger.Info(fmt.Sprintf(fmtStr, args...))
}

func (logg *polvoLogger) PrintError(fmtStr string, args ...any) {
	logg.logger.Warn(fmt.Sprintf(fmtStr, args...))
}

func (logg *polvoLogger) getLogFileName() string {
	return filepath.Join(logg.logPath, "service.log")
}

type CustomLoggerForOpenSearch struct {
	Logger *zap.Logger
}

func (c *CustomLoggerForOpenSearch) Printf(format string, v ...interface{}) {
	c.Logger.Sugar().Infof(format, v...)
}

func (c *CustomLoggerForOpenSearch) Debugf(format string, v ...interface{}) {
	c.Logger.Sugar().Debugf(format, v...)
}

func (c *CustomLoggerForOpenSearch) LogRoundTrip(req *http.Request, res *http.Response, err error, start time.Time, d time.Duration) error {
	c.Logger.Sugar().Infof("RoundTrip: %s %s %s %v %v", req.Method, req.URL, res.Status, err, d)
	// TODO: request/response body logging
	return nil
}

func (c *CustomLoggerForOpenSearch) RequestBodyEnabled() bool {
	return true
}

func (c *CustomLoggerForOpenSearch) ResponseBodyEnabled() bool {
	return true
}
