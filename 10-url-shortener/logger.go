/*
logger for a history of request and errors
*/
package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Level string //custom type for log levels

const (
	DEBUG Level = "DEBUG"
	INFO  Level = "INFO"
	WARN  Level = "WARN"
	ERROR Level = "ERROR"
)

// if converted to json it would take those fields
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     Level     `json:"level"`
	Message   string    `json:"message"`
	Service   string    `json:"service"` //service that generated the log entry
}

type Logger struct {
	serviceName string
}

func NewLogger(serviceName string) *Logger {
	return &Logger{
		serviceName: serviceName,
	}
}

func (l *Logger) Log(level Level, message string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Service:   l.serviceName,
	}
	stdout, _ := json.Marshal(entry) //convert log entry to json
	fmt.Println(string(stdout))      //print log entry to console (or write to a file)
}
