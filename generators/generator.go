// Package generators provides functions to apply themes by processing wallpapers,
// including expanding glob patterns and converting Lottie animations and GLSL shaders to video formats.
package generators

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/AlexMacocian/rainbeau/converters"
	rainbeau "github.com/AlexMacocian/rainbeau/internal"
)

var generatorLogger = rainbeau.GetLogger("generator", slog.LevelInfo)

func ApplyTheme(theme *rainbeau.Theme, outputDir string, wallpaperDir string) error {
	rainbeau.NotifyInfo("Rainbeau", fmt.Sprintf("Applying theme %s", theme.Name))

	globErr := expandWallpaperGlobs(theme, wallpaperDir)
	if globErr != nil {
		generatorLogger.Error("Failed to expand wallpaper globs", "error", globErr)
		return globErr
	}

	generatorLogger.Debug("Expanded theme", "name", theme)
	if len(theme.Wallpapers.Lotties) > 0 {
		renderedGifs := converters.ConvertLotties(theme.Wallpapers.Lotties, wallpaperDir, theme.Colors.Bg0, theme.Colors.Border)
		appendResult := append(theme.Wallpapers.Videos, renderedGifs...)
		generatorLogger.Info("Converted Lottie animations to MP4s", "lottieFiles", renderedGifs)
		theme.Wallpapers.Videos = appendResult
	}

	if len(theme.Wallpapers.Shaders) > 0 {
		renderedShaders := converters.ConvertShaders(theme.Wallpapers.Shaders, wallpaperDir, theme.Colors.Bg0, theme.Colors.Border)
		appendResult := append(theme.Wallpapers.Videos, renderedShaders...)
		generatorLogger.Info("Converted GLSL shaders to MP4s", "shaderFiles", renderedShaders)
		theme.Wallpapers.Videos = appendResult
	}

	return nil
}

func expandWallpaperGlobs(theme *rainbeau.Theme, wallpapersDir string) error {
	var err error

	theme.Wallpapers.Images, err = expandGlobs(theme.Wallpapers.Images, wallpapersDir)
	if err != nil {
		return err
	}

	theme.Wallpapers.Videos, err = expandGlobs(theme.Wallpapers.Videos, wallpapersDir)
	if err != nil {
		return err
	}

	theme.Wallpapers.Lotties, err = expandGlobs(theme.Wallpapers.Lotties, wallpapersDir)
	if err != nil {
		return err
	}

	return nil
}

func expandGlobs(patterns []string, baseDir string) ([]string, error) {
	var result []string

	for _, pattern := range patterns {
		fullPattern := filepath.Join(baseDir, pattern)

		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 {
			result = append(result, fullPattern)
			continue
		}

		result = append(result, matches...)
	}

	return result, nil
}
