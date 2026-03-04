package gorp

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ConfigureLog(logLevel *string) {

	// Set up the logger with the specified level
	var level slog.Level
	switch strings.ToLower(*logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// check logs folder exists
	newpath := filepath.Join(".", "logs")
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		err := fmt.Errorf("Error creating logs folder\n")

		if err != nil {
			fmt.Println("Cascading errors, please check file permissions")
			fmt.Println("Instance is highly likely to malfunction!")
		}
	}

	// create log file name
	datetime := fmt.Sprintf("%s", time.Now().UTC())
	cleaned_datetime := datetime[:len(datetime)-9]

	random_prefix := rand.Text()[:8]

	fileName := strings.ReplaceAll(fmt.Sprintf("%s-%s.log", random_prefix, cleaned_datetime), " ", "")
	logFile, err := os.OpenFile(filepath.Join(".", "logs", fileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	if err != nil {
		err := fmt.Errorf("Error creating log file: %s", fileName)

		if err != nil {
			fmt.Println("Cascading errors, please check file permissions")
			fmt.Println("Instance is highly likely to malfunction!")
		}
	} else {
		fmt.Printf("Log file: %s", fileName)
	}

	// Create a logger with the desired level
	logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: level,
	}))

	// Set the logger as the default
	slog.SetDefault(logger)
}
