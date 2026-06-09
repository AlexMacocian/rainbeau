package main

import (
	"log/slog"
	"os"
)

// loaderLogger is a global slog.Logger instance for the loader component.
var loaderLogger = getLogger("loader", slog.LevelInfo)

// mainLogger is a global slog.Logger instance for the main component.
var mainLogger = getLogger("main", slog.LevelInfo)

// mainLogger is a global slog.Logger instance for the main component.
var generatorLogger = getLogger("generator", slog.LevelInfo)

// getLogger creates a new slog.Logger with the specified component name and log level.
func getLogger(component string, level slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler).With("component", component)
}
