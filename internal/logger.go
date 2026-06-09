// Package internal provides internal utilities for the application, including logging functionality.
package internal

import (
	"log/slog"
	"os"
)

// LoaderLogger is a global slog.Logger instance for the loader component.
var LoaderLogger = GetLogger("loader", slog.LevelInfo)

// MainLogger is a global slog.Logger instance for the main component.
var MainLogger = GetLogger("main", slog.LevelInfo)

// GeneratorLogger is a global slog.Logger instance for the generator component.
var GeneratorLogger = GetLogger("generator", slog.LevelInfo)

// ConverterLogger is a global slog.Logger instance for the converter component.
var ConverterLogger = GetLogger("converter", slog.LevelInfo)

// GetLogger creates a new slog.Logger with the specified component name and log level.
func GetLogger(component string, level slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler).With("component", component)
}
