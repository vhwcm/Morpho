package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Level string

const (
	LevelInfo  Level = "INFO"
	LevelError Level = "ERROR"
	LevelDebug Level = "DEBUG"
)

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     Level                  `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

var (
	logFile *os.File
	LogDir  = ".morpho/logs"
	LogName = "app.log"
)

func Init() error {
	if _, err := os.Stat(LogDir); os.IsNotExist(err) {
		if err := os.MkdirAll(LogDir, 0755); err != nil {
			return fmt.Errorf("falha ao criar diretório de logs: %w", err)
		}
	}

	path := filepath.Join(LogDir, LogName)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo de log: %w", err)
	}
	logFile = f
	return nil
}

func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

func log(level Level, msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
		Fields:    fields,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao serializar log: %v\n", err)
		return
	}

	if logFile != nil {
		fmt.Fprintln(logFile, string(data))
	}
}

func Info(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	log(LevelInfo, msg, f)
}

func Error(msg string, err error, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	if err != nil {
		f["error"] = err.Error()
	}
	log(LevelError, msg, f)
}

func Debug(msg string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	log(LevelDebug, msg, f)
}

func GetLogPath() string {
	return filepath.Join(LogDir, LogName)
}

func RecoverPanic() {
	if r := recover(); r != nil {
		Error("PANIC RECOVERY", fmt.Errorf("%v", r))
	}
}
