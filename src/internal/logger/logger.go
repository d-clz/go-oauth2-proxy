package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	currentLevel Level = INFO
	logger       *log.Logger
)

func Init(levelStr string) {
	logger = log.New(os.Stdout, "", 0)
	SetLevel(levelStr)
}

func SetLevel(levelStr string) {
	switch strings.ToLower(levelStr) {
	case "debug":
		currentLevel = DEBUG
	case "info":
		currentLevel = INFO
	case "warn":
		currentLevel = WARN
	case "error":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}
}

func formatMessage(level string, msg string, keysAndValues ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	result := timestamp + " [" + level + "] " + msg

	// Add key-value pairs
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			result += " " + keysAndValues[i].(string) + "=" + format(keysAndValues[i+1])
		}
	}
	return result
}

func format(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return fmt.Sprint(val)
	}
}

func Debug(msg string, keysAndValues ...interface{}) {
	if currentLevel <= DEBUG {
		logger.Println(formatMessage("DEBUG", msg, keysAndValues...))
	}
}

func Info(msg string, keysAndValues ...interface{}) {
	if currentLevel <= INFO {
		logger.Println(formatMessage("INFO", msg, keysAndValues...))
	}
}

func Warn(msg string, keysAndValues ...interface{}) {
	if currentLevel <= WARN {
		logger.Println(formatMessage("WARN", msg, keysAndValues...))
	}
}

func Error(msg string, keysAndValues ...interface{}) {
	if currentLevel <= ERROR {
		logger.Println(formatMessage("ERROR", msg, keysAndValues...))
	}
}

func Fatal(msg string, keysAndValues ...interface{}) {
	logger.Println(formatMessage("FATAL", msg, keysAndValues...))
	os.Exit(1)
}
