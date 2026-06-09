// Package generators provides functions to apply themes by processing wallpapers,
// including expanding glob patterns and converting Lottie animations and GLSL shaders to video formats.
package generators

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

	reloadThemeServices(theme)

	if len(errors) > 0 {
		rainbeau.NotifyError(fmt.Sprintf("Theme '%s' applied with %d error(s).", theme.Name, len(errors)))
		generatorLogger.Error("Theme applied with generator errors", "errors", strings.Join(errors, "; "))
		return nil
	}

	rainbeau.NotifySuccess(fmt.Sprintf("Theme '%s' applied successfully.", theme.Name))
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

func reloadThemeServices(theme *rainbeau.Theme) {
	restartWallpaperCycler()
	reloadKitty()
	reloadNeovim(theme)

	runCommand("hyprctl", "reload")
	runCommand("killall", "-SIGUSR2", "waybar")
	time.Sleep(300 * time.Millisecond)
	if !isProcessRunning("waybar") {
		runDetached("waybar")
	}

	runCommand("killall", "dunst")
	runCommand("gsettings", "set", "org.gnome.desktop.interface", "color-scheme", theme.Gtk.ColorScheme)
	runCommand("gsettings", "set", "org.gnome.desktop.interface", "gtk-theme", theme.Gtk.Theme)

	runCommand("killall", "hyprpaper")
	runDetached("hyprpaper")
}

func restartWallpaperCycler() {
	runCommand("pkill", "-f", "wallpaper-cycler")
	time.Sleep(300 * time.Millisecond)
	runDetached("bash", "-c", "~/.config/hypr/scripts/wallpaper-cycler.sh &")
}

func reloadKitty() {
	sockets, err := filepath.Glob("/tmp/kitty-socket-*")
	if err != nil {
		generatorLogger.Error("Failed to find Kitty sockets", "error", err)
		return
	}

	for _, socket := range sockets {
		runCommand("kitten", "@", "--to", "unix:"+socket, "set-colors", "--all", "--configured", "~/.config/kitty/kitty.conf")
	}
}

func reloadNeovim(theme *rainbeau.Theme) {
	home, err := os.UserHomeDir()
	if err == nil {
		if err := os.RemoveAll(filepath.Join(home, ".cache", "nvim", "catppuccin")); err != nil {
			generatorLogger.Error("Failed to remove catppuccin cache", "error", err)
		}
	}

	uid := os.Getenv("UID")
	if uid == "" {
		loginUID, err := os.ReadFile("/proc/self/loginuid")
		if err == nil {
			uid = strings.TrimSpace(string(loginUID))
		}
	}
	if uid == "" {
		return
	}

	nvimDir := filepath.Join("/run/user", uid)
	if info, err := os.Stat(nvimDir); err != nil || !info.IsDir() {
		return
	}

	cmd := ":lua package.loaded['plugins.colorscheme'] = nil; require('catppuccin').setup(require('plugins.colorscheme')[1].opts); vim.cmd.colorscheme('catppuccin')<CR>"
	if theme.Nvim.ColorScheme != "" {
		cmd = fmt.Sprintf(":silent! colorscheme %s<CR>", theme.Nvim.ColorScheme)
	}

	sockets, err := filepath.Glob(filepath.Join(nvimDir, "nvim.*"))
	if err != nil {
		generatorLogger.Error("Failed to find Neovim sockets", "error", err)
		return
	}

	for _, socket := range sockets {
		runCommand("nvim", "--server", socket, "--remote-send", cmd)
	}
}

func runCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		generatorLogger.Debug("Command failed to start", "command", name, "args", args, "error", err)
		return
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			generatorLogger.Debug("Command failed", "command", name, "args", args, "error", err)
		}
	case <-time.After(5 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			generatorLogger.Debug("Failed to kill timed-out command", "command", name, "error", err)
		}
		<-done
	}
}

func runDetached(name string, args ...string) {
	if err := exec.Command(name, args...).Start(); err != nil {
		generatorLogger.Debug("Detached command failed to start", "command", name, "args", args, "error", err)
	}
}

func isProcessRunning(name string) bool {
	cmd := exec.Command("pgrep", "-x", name)
	if err := cmd.Start(); err != nil {
		return false
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		return err == nil
	case <-time.After(2 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			generatorLogger.Debug("Failed to kill timed-out pgrep", "process", name, "error", err)
		}
		<-done
		return false
	}
}
