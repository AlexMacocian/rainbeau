package main

import "path/filepath"

func applyTheme(theme *Theme, outputDir string, wallpaperDir string) error {
	globErr := expandWallpaperGlobs(theme, wallpaperDir)
	if globErr != nil {
		generatorLogger.Error("Failed to expand wallpaper globs", "error", globErr)
		return globErr
	}

	generatorLogger.Debug("Expanded theme", "name", theme)
	if len(theme.Wallpapers.Lotties) > 0 {
		renderedGifs := convertLotties(theme.Wallpapers.Lotties, wallpaperDir, theme.Colors.Bg0, theme.Colors.Border)
		appendResult := append(theme.Wallpapers.Videos, renderedGifs...)
		generatorLogger.Info("Converted Lottie animations to MP4s", "lottieFiles", renderedGifs)
		theme.Wallpapers.Videos = appendResult
	}

	if len(theme.Wallpapers.Shaders) > 0 {
		renderedShaders := convertShaders(theme.Wallpapers.Shaders, wallpaperDir, theme.Colors.Bg0, theme.Colors.Border)
		appendResult := append(theme.Wallpapers.Videos, renderedShaders...)
		generatorLogger.Info("Converted GLSL shaders to MP4s", "shaderFiles", renderedShaders)
		theme.Wallpapers.Videos = appendResult
	}

	return nil
}

func expandWallpaperGlobs(theme *Theme, wallpapersDir string) error {
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
