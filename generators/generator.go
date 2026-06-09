// Package generators provides functions to apply themes by processing wallpapers,
// including expanding glob patterns and converting Lottie animations and GLSL shaders to video formats.
package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlexMacocian/rainbeau/converters"
	rainbeau "github.com/AlexMacocian/rainbeau/internal"
)

var generatorLogger = rainbeau.GeneratorLogger

type Generator interface {
	Name() string
	OutputPath() string
	Generate(theme *rainbeau.Theme, wallpapersDir string) (string, error)
}

func ApplyTheme(theme *rainbeau.Theme, outputDir string, wallpaperDir string) error {
	rainbeau.NotifyInfo(fmt.Sprintf("Applying theme %s", theme.Name))

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

	generators := []Generator{
		HyprlandGenerator{},
		HyprpaperGenerator{},
		DunstGenerator{},
		GtkSettingsGenerator{},
		GtkCSSGenerator{},
		KittyGenerator{},
		WofiStyleGenerator{},
		WaybarConfigGenerator{},
		WaybarStyleGenerator{},
		HyprlockGenerator{},
		WallpaperCyclerGenerator{},
		WallpaperSwitchGenerator{},
		TemperatureScriptGenerator{},
		BluetoothScriptGenerator{},
		GpuScriptGenerator{},
		HyprchatGenerator{},
		HyprtoolkitGenerator{},
		OmniLauncherConfigGenerator{},
		QuickVisorThemeGenerator{},
		FirefoxGenerator{},
		FirefoxThemeGenerator{},
		FirefoxPrefsGenerator{},
		FirefoxContentGenerator{},
		VscodeSettingsGenerator{},
		NvimColorschemeGenerator{},
	}

	var errors []string
	for _, gen := range generators {
		outPath := filepath.Join(outputDir, gen.OutputPath())
		if err := ensureOutputDirectory(outputDir, filepath.Dir(outPath)); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", gen.Name(), err))
			rainbeau.NotifyError(fmt.Sprintf("Generator '%s' failed: %v", gen.Name(), err))
			continue
		}

		content, err := gen.Generate(theme, wallpaperDir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", gen.Name(), err))
			rainbeau.NotifyError(fmt.Sprintf("Generator '%s' failed: %v", gen.Name(), err))
			continue
		}

		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", gen.Name(), err))
			rainbeau.NotifyError(fmt.Sprintf("Generator '%s' failed: %v", gen.Name(), err))
			continue
		}

		generatorLogger.Info("Generated config", "generator", gen.Name(), "path", outPath)
	}

	if err := chmodScripts(filepath.Join(outputDir, ".config/hypr/scripts")); err != nil {
		errors = append(errors, fmt.Sprintf("scripts: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("generator errors: %s", strings.Join(errors, "; "))
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
		if !strings.ContainsAny(pattern, "*?") {
			result = append(result, pattern)
			continue
		}

		dir := filepath.Dir(pattern)
		filePattern := filepath.Base(pattern)
		searchDir := baseDir
		if dir != "." && dir != "" {
			searchDir = filepath.Join(baseDir, dir)
		}

		if info, err := os.Stat(searchDir); err != nil || !info.IsDir() {
			continue
		}

		matches, err := filepath.Glob(filepath.Join(searchDir, filePattern))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			rel, err := filepath.Rel(baseDir, match)
			if err != nil {
				return nil, err
			}
			result = append(result, rel)
		}
	}

	return result, nil
}

func ensureOutputDirectory(outputDir string, outDir string) error {
	pathToCheck := outDir
	for pathToCheck != "" && strings.HasPrefix(pathToCheck, outputDir) && pathToCheck != outputDir {
		if info, err := os.Stat(pathToCheck); err == nil && !info.IsDir() {
			if err := os.Remove(pathToCheck); err != nil {
				return err
			}
			break
		}
		pathToCheck = filepath.Dir(pathToCheck)
	}
	return os.MkdirAll(outDir, 0o755)
}

func chmodScripts(scriptDir string) error {
	entries, err := filepath.Glob(filepath.Join(scriptDir, "*.sh"))
	if err != nil {
		return err
	}
	for _, script := range entries {
		if err := os.Chmod(script, 0o755); err != nil {
			return err
		}
	}
	return nil
}
