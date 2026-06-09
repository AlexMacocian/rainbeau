package main

import (
	"flag"
	"os"
	"path/filepath"
)

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

	outputDir, loadErr := getOuputDir()
	if (loadErr != nil) || (outputDir == nil) {
		mainLogger.Error("Failed to get output directory", "error", loadErr)
		os.Exit(1)
	}

	wallpaperDir := getWallpaperDir(args[0])
	mainLogger.Info("Loading theme", "name", theme.Name)
	mainLogger.Info("Output directory", "path", *outputDir)
	mainLogger.Info("Wallpaper directory", "path", wallpaperDir)

	applyErr := applyTheme(theme, *outputDir, wallpaperDir)

	if applyErr != nil {
		mainLogger.Error("Failed to apply theme", "error", applyErr)
		os.Exit(1)
	}
}

func getOuputDir() (*string, error) {
	outputDirDefault, err := os.UserHomeDir()
	if err != nil {
		mainLogger.Error("Failed to get user home directory", "error", err)
		return nil, err
	}

	outputDir := flag.String("output-dir", outputDirDefault, "output directory")
	flag.Parse()

	absOutputDir, err := filepath.Abs(*outputDir)
	if (err != nil) || (absOutputDir == "") {
		mainLogger.Error("Failed to resolve output directory", "error", err)
		return nil, err
	}

	return outputDir, nil
}

func getWallpaperDir(themePath string) string {
	return filepath.Dir(themePath)
}
