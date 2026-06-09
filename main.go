// Package main is the entry point of the Rainbeau application. It parses command-line arguments, loads the theme, and applies it to the system.
package main

import (
	"os"
	"path/filepath"

	"github.com/AlexMacocian/rainbeau/generators"
	rainbeau "github.com/AlexMacocian/rainbeau/internal"
)

var mainLogger = rainbeau.MainLogger

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		mainLogger.Error("Usage: rainbeau <theme.json> [--output-dir <dir>]")
		mainLogger.Error("       rainbeau theme.json --output-dir ~/.config")
		os.Exit(1)
	}

	theme, parseErr := loadTheme(args[0])
	if parseErr != nil {
		mainLogger.Error("Failed to load theme", "error", parseErr)
		os.Exit(1)
	}

	outputDir, loadErr := getOuputDir(args)
	if loadErr != nil {
		mainLogger.Error("Failed to get output directory", "error", loadErr)
		os.Exit(1)
	}

	wallpaperDir := getWallpaperDir(args[0])
	mainLogger.Info("Loading theme", "name", theme.Name)
	mainLogger.Info("Output directory", "path", outputDir)
	mainLogger.Info("Wallpaper directory", "path", wallpaperDir)

	applyErr := generators.ApplyTheme(theme, outputDir, wallpaperDir)

	if applyErr != nil {
		mainLogger.Error("Failed to apply theme", "error", applyErr)
		os.Exit(1)
	}
}

func getOuputDir(args []string) (string, error) {
	outputDir, err := os.UserHomeDir()
	if err != nil {
		mainLogger.Error("Failed to get user home directory", "error", err)
		return "", err
	}

	for i, arg := range args {
		if arg == "--output-dir" && i+1 < len(args) {
			outputDir = args[i+1]
			break
		}
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if (err != nil) || (absOutputDir == "") {
		mainLogger.Error("Failed to resolve output directory", "error", err)
		return "", err
	}

	return absOutputDir, nil
}

func getWallpaperDir(themePath string) string {
	return filepath.Dir(themePath)
}
