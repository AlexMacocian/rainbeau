package internal

import (
	"encoding/json"
	"os"
)

// LoadTheme reads a JSON theme file from the specified path, parses it into a Theme struct, and returns it.
func LoadTheme(themePath string) (*Theme, error) {
	themeFile, readErr := os.ReadFile(themePath)
	if readErr != nil {
		LoaderLogger.Error("Error opening theme file", "error", readErr)
		return nil, readErr
	}

	LoaderLogger.Debug("Theme file", "path", themePath, "size", len(themeFile))
	var data Theme
	parseErr := json.Unmarshal(themeFile, &data)
	if parseErr != nil {
		LoaderLogger.Error("Error parsing theme file", "error", parseErr)
		return nil, parseErr
	}

	LoaderLogger.Debug("Parsed theme data", "data", data)
	return &data, nil
}
