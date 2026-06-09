package main

import (
	"encoding/json"
	"os"
)

// loadTheme reads a JSON theme file from the specified path, parses it into a Theme struct, and returns it.
func loadTheme(themePath string) (*Theme, error) {
	themeFile, readErr := os.ReadFile(themePath)
	if readErr != nil {
		loaderLogger.Error("Error opening theme file", "error", readErr)
		return nil, readErr
	}

	loaderLogger.Debug("Theme file", "path", themePath, "size", len(themeFile))
	var data Theme
	parseErr := json.Unmarshal(themeFile, &data)
	if parseErr != nil {
		loaderLogger.Error("Error parsing theme file", "error", parseErr)
		return nil, parseErr
	}

	loaderLogger.Debug("Parsed theme data", "data", data)
	return &data, nil
}
