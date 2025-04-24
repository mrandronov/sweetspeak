package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LogLevel int

const (
	INFO  LogLevel = 0
	ERROR LogLevel = 1
	WARN  LogLevel = 2
	DEBUG LogLevel = 3
)

var (
	DefaultTimeFormat = time.RFC3339
	DefaultOut        = os.Stdout
	DefaultLogDir     = "logs"

	LogNames = map[LogLevel]string{
		INFO:  "INFO",
		ERROR: "EROR",
		WARN:  "WARN",
		DEBUG: "DBUG",
	}
	LogColor = map[LogLevel]string{
		INFO:  Cyan,
		ERROR: Red,
		WARN:  Yellow,
		DEBUG: Purple,
	}
)

type (
	Logger struct {
		sync.Mutex
		Level      LogLevel
		TimeFormat string
		LogFormat  string
		WriteCh    chan LogMsg
		Config     Config
		LogDir     string
		Loggers    map[io.Writer]*log.Logger
	}

	Config struct {
		Level      LogLevel
		Console    bool
		File       string
		Colorize   bool
		MaxSize    int
		MaxAge     int
		TimeFormat string
	}

	LogMsg struct {
		level     LogLevel
		timestamp time.Time
		msg       string
	}
)

func New(config Config) *Logger {
	l := &Logger{
		Level:     config.Level,
		LogFormat: "%s %s %s",
		WriteCh:   make(chan LogMsg),
		Config:    config,
		LogDir:    DefaultLogDir,
		Loggers:   make(map[io.Writer]*log.Logger),
	}

	if config.TimeFormat == "" {
		l.TimeFormat = DefaultTimeFormat
	}

	if config.Console {
		l.addWriter(os.Stdout)
	}

	if config.File != "" {
		rotateLogger := &lumberjack.Logger{
			Filename:   l.LogDir + "/" + config.File,
			MaxSize:    config.MaxSize,
			MaxAge:     config.MaxAge,
			MaxBackups: 0,
			Compress:   false,
		}
		l.addWriter(rotateLogger)
	}

	if len(l.Loggers) == 0 {
		l.addWriter(DefaultOut)
	}

	go l.WriteLogs()

	return l
}

func (l *Logger) addWriter(w io.Writer) {
	l.Lock()
	defer l.Unlock()

	l.Loggers[w] = log.New(w, "", 0)
}

func (l *Logger) SetLevel(level LogLevel) {
	l.Lock()
	defer l.Unlock()

	l.Config.Level = level
}

func (l *Logger) WriteLogs() {
	for logMsg := range l.WriteCh {
		logString := l.FormatLog(logMsg)
		l.log(logString)
	}
}

func (l *Logger) log(logString string) {
	for _, logger := range l.Loggers {
		// Eat errors for now (yum!)
		_, err := logger.Writer().Write([]byte(logString))
		if err != nil {
			fmt.Printf("error writing log: %v", err)
		}
	}
}

func (l *Logger) FormatLog(logMessage LogMsg) string {
	timestamp := logMessage.timestamp.Format(l.TimeFormat)
	levelName := LogNames[logMessage.level]
	msgStr := logMessage.msg

	if l.Config.Colorize {
		levelName = colorize(LogColor[logMessage.level], levelName)
	}

	return fmt.Sprintf(l.LogFormat, timestamp, levelName, msgStr) + "\n"
}

func (l *Logger) writeMsg(level LogLevel, format string, args ...interface{}) {
	if level > l.Level {
		return
	}

	logMsg := LogMsg{
		level:     level,
		timestamp: time.Now(),
		msg:       fmt.Sprintf(format, args...),
	}

	select {
	case l.WriteCh <- logMsg:
	case <-time.NewTimer(time.Second).C:
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.writeMsg(INFO, format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.writeMsg(DEBUG, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.writeMsg(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.writeMsg(ERROR, format, args...)
}

var DefaultLogger *Logger

func init() {
	DefaultLogger = New(
		Config{
			Level:    DEBUG,
			File:     "sweetspeak.log",
			Console:  true,
			Colorize: true,
			MaxSize:  10,
			MaxAge:   1,
		})
}

func Debug(format string, v ...interface{}) {
	DefaultLogger.Debug(format, v...)
}

func Info(format string, v ...interface{}) {
	DefaultLogger.Info(format, v...)
}

func Warn(format string, v ...interface{}) {
	DefaultLogger.Warn(format, v...)
}

func Error(format string, v ...interface{}) {
	DefaultLogger.Error(format, v...)
}

func SetGlobalLevel(level LogLevel) {
	DefaultLogger.SetLevel(level)
}

func SetGlobalFile(fileName string) {
        // Reinit with new config
        currentLevel := DefaultLogger.Level
	DefaultLogger = New(
		Config{
			Level:    currentLevel,
			File:     fileName,
			Console:  true,
			Colorize: true,
			MaxSize:  10,
			MaxAge:   1,
		})
}

func SetConsoleOutput(console bool) {
	DefaultLogger = New(
		Config{
			Level:    DefaultLogger.Level,
			File:     DefaultLogger.Config.File,
			Console:  console,
			Colorize: true,
			MaxSize:  10,
			MaxAge:   1,
		})
}

//************* COLORS **************

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

func colorize(color, input string) string {
	return color + input + Reset
}
