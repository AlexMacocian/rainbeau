// Package main is the entry point of the Rainbeau application. It parses command-line arguments,
// loads or selects a theme, and applies it to the system.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlexMacocian/rainbeau/generators"
	rainbeau "github.com/AlexMacocian/rainbeau/internal"
	"github.com/AlexMacocian/rainbeau/selector"
)

var mainLogger = rainbeau.MainLogger

func main() {
	args := os.Args[1:]
	if len(args) < 1 || args[0] != "select" {
		printUsage()
		os.Exit(1)
	}

	if err := runSelect(args[1:]); err != nil {
		mainLogger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}

// runSelect handles the `select` subcommand in both modes:
//   - rainbeau select <theme.json> [--output-dir <dir>]            apply directly
//   - rainbeau select [--themes-dir <dir>] [--output-dir <dir>]    open the visual picker
func runSelect(args []string) error {
	options, err := parseSelectArgs(args)
	if err != nil {
		printUsage()
		return err
	}

	themePath := options.themePath
	if themePath == "" {
		chosen, selectErr := selector.Run(options.themesDir)
		if selectErr != nil {
			return selectErr
		}
		if chosen == "" {
			mainLogger.Info("No theme selected")
			return nil
		}
		themePath = chosen
	}

	return applyTheme(themePath, options.outputDir)
}

func applyTheme(themePath string, outputDir string) error {
	theme, parseErr := rainbeau.LoadTheme(themePath)
	if parseErr != nil {
		mainLogger.Error("Failed to load theme", "error", parseErr)
		return parseErr
	}

	resolvedOutput, outputErr := resolveOutputDir(outputDir)
	if outputErr != nil {
		mainLogger.Error("Failed to get output directory", "error", outputErr)
		return outputErr
	}

	wallpaperDir := filepath.Dir(themePath)
	mainLogger.Info("Loading theme", "name", theme.Name)
	mainLogger.Info("Output directory", "path", resolvedOutput)
	mainLogger.Info("Wallpaper directory", "path", wallpaperDir)

	if err := generators.ApplyTheme(theme, resolvedOutput, wallpaperDir); err != nil {
		return err
	}

	if err := selector.SaveCurrent(theme); err != nil {
		mainLogger.Warn("Failed to record current theme", "error", err)
	}
	return nil
}

type selectOptions struct {
	themePath string
	themesDir string
	outputDir string
}

// parseSelectArgs extracts the optional positional theme path and the
// --output-dir / --themes-dir flags. The themes directory defaults to
// ~/.config/rainbeau/themes when not provided.
func parseSelectArgs(args []string) (selectOptions, error) {
	options := selectOptions{}

	themesDir, err := defaultThemesDir()
	if err != nil {
		return options, err
	}
	options.themesDir = themesDir

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--output-dir":
			if i+1 >= len(args) {
				return options, fmt.Errorf("missing value for %s", arg)
			}
			options.outputDir = args[i+1]
			i++
		case "--themes-dir":
			if i+1 >= len(args) {
				return options, fmt.Errorf("missing value for %s", arg)
			}
			options.themesDir = args[i+1]
			i++
		default:
			if options.themePath != "" {
				return options, fmt.Errorf("unexpected argument %q", arg)
			}
			options.themePath = arg
		}
	}

	return options, nil
}

func defaultThemesDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "rainbeau", "themes"), nil
}

func resolveOutputDir(outputDir string) (string, error) {
	if outputDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		outputDir = home
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil || absOutputDir == "" {
		return "", err
	}

	return absOutputDir, nil
}

func printUsage() {
	mainLogger.Error("Usage: rainbeau select <theme.json> [--output-dir <dir>]")
	mainLogger.Error("       rainbeau select [--themes-dir <dir>] [--output-dir <dir>]")
}
