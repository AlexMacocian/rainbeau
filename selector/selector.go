// Package selector implements the visual theme picker. It discovers available
// theme files, renders preview thumbnails, and drives a quickshell-based UI that
// lets the user pick a theme. The chosen theme path is returned to the caller so
// the regular theme-application pipeline can act on it.
package selector

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	rainbeau "github.com/AlexMacocian/rainbeau/internal"
)

//go:embed qml/*.qml
var qmlFS embed.FS

var selectorLogger = rainbeau.GetLogger("selector", slog.LevelInfo)

const (
	thumbWidth  = 640
	thumbHeight = 360
)

// ErrNoThemes is returned when the themes directory contains no theme files.
var ErrNoThemes = errors.New("no themes found")

// ErrQuickshellMissing is returned when the quickshell binary is not available.
var ErrQuickshellMissing = errors.New("quickshell is not installed")

// manifestTheme is a single entry passed to the QML picker. Thumbnail is the
// palette placeholder rendered up front so the UI can show instantly; Image is
// the real preview rendered in the background and appears on disk later (empty
// when the theme has no usable image source).
type manifestTheme struct {
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	Thumbnail string            `json:"thumbnail"`
	Image     string            `json:"image"`
	Font      string            `json:"font"`
	Colors    map[string]string `json:"colors"`
}

type manifest struct {
	Themes         []manifestTheme `json:"themes"`
	ActiveBorder   string          `json:"activeBorder"`
	InactiveBorder string          `json:"inactiveBorder"`
}

const (
	defaultActiveBorder   = "#ffffff"
	defaultInactiveBorder = "#555555"
)

// SaveCurrent records the colors of the just-applied theme so the picker can use
// the active/inactive surface colors of the current theme for its selection
// highlight on the next run.
func SaveCurrent(theme *rainbeau.Theme) error {
	cacheDir, err := ensureCacheDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]any{"colors": paletteMap(theme.Colors)}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, "current.json"), data, 0o644)
}

// loadCurrentBorders returns the active and inactive border colors derived from
// the currently applied theme: its primary background (bg0) and inactive color.
// Sensible defaults are returned when no current theme has been recorded.
func loadCurrentBorders(cacheDir string) (active string, inactive string) {
	active, inactive = defaultActiveBorder, defaultInactiveBorder

	data, err := os.ReadFile(filepath.Join(cacheDir, "current.json"))
	if err != nil {
		return active, inactive
	}

	var current struct {
		Colors map[string]string `json:"colors"`
	}
	if err := json.Unmarshal(data, &current); err != nil {
		return active, inactive
	}
	if v := current.Colors["bg0"]; v != "" {
		active = v
	}
	if v := current.Colors["inactive"]; v != "" {
		inactive = v
	}
	return active, inactive
}

// Run discovers themes under themesDir, renders thumbnails, launches the
// quickshell picker, and returns the absolute path of the chosen theme file.
// An empty string with a nil error means the user cancelled the selection.
func Run(themesDir string) (string, error) {
	quickshell, lookErr := exec.LookPath("quickshell")
	if lookErr != nil {
		quickshell, lookErr = exec.LookPath("qs")
		if lookErr != nil {
			return "", ErrQuickshellMissing
		}
	}

	cacheDir, cacheErr := ensureCacheDir()
	if cacheErr != nil {
		return "", cacheErr
	}

	entries, discoverErr := discoverThemes(themesDir)
	if discoverErr != nil {
		return "", discoverErr
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("%w in %s", ErrNoThemes, themesDir)
	}

	man, jobs := buildManifest(entries, filepath.Join(cacheDir, "thumbs"))
	man.ActiveBorder, man.InactiveBorder = loadCurrentBorders(cacheDir)

	manifestPath := filepath.Join(cacheDir, "manifest.json")
	if err := writeManifest(man, manifestPath); err != nil {
		return "", err
	}

	qmlDir := filepath.Join(cacheDir, "qml")
	if err := extractQML(qmlDir); err != nil {
		return "", err
	}

	resultPath := filepath.Join(cacheDir, "selection")
	_ = os.Remove(resultPath)

	// Render the real preview images in the background so the picker (already
	// populated with palette placeholders) appears instantly. The context is
	// cancelled once the picker exits to stop any in-flight ffmpeg work.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go generateThumbnails(ctx, jobs)

	cmd := exec.Command(quickshell, "-p", filepath.Join(qmlDir, "shell.qml"))
	cmd.Env = append(os.Environ(),
		"RAINBEAU_MANIFEST="+manifestPath,
		"RAINBEAU_RESULT="+resultPath,
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	selectorLogger.Info("Launching theme picker", "themes", len(entries), "previews", len(jobs))
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("quickshell exited with error: %w", err)
	}

	selection, readErr := os.ReadFile(resultPath)
	if readErr != nil {
		// No result file means the picker was dismissed without a choice.
		selectorLogger.Info("Selection cancelled")
		return "", nil
	}

	chosen := strings.TrimSpace(string(selection))
	return chosen, nil
}

type themeEntry struct {
	theme *rainbeau.Theme
	path  string
	dir   string
}

// discoverThemes loads every *.json theme file directly inside themesDir.
func discoverThemes(themesDir string) ([]themeEntry, error) {
	absDir, err := filepath.Abs(themesDir)
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(absDir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)

	var entries []themeEntry
	for _, match := range matches {
		theme, loadErr := rainbeau.LoadTheme(match)
		if loadErr != nil {
			selectorLogger.Error("Skipping unreadable theme", "path", match, "error", loadErr)
			continue
		}
		entries = append(entries, themeEntry{theme: theme, path: match, dir: filepath.Dir(match)})
	}

	return entries, nil
}

// thumbJob describes deferred work to render a real preview image for a theme.
type thumbJob struct {
	name   string
	source string
	dst    string
}

// buildManifest renders the fast palette placeholders synchronously and returns
// the manifest plus the list of deferred jobs to render real preview images.
func buildManifest(entries []themeEntry, thumbsDir string) (manifest, []thumbJob) {
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		selectorLogger.Error("Failed to create thumbnails directory", "error", err)
	}

	man := manifest{Themes: make([]manifestTheme, 0, len(entries))}
	var jobs []thumbJob

	for i, entry := range entries {
		slug := slugify(entry.path, i)
		palettePath := filepath.Join(thumbsDir, slug+".png")
		if err := renderPaletteThumbnail(entry.theme.Colors, palettePath); err != nil {
			selectorLogger.Error("Failed to render palette thumbnail", "theme", entry.theme.Name, "error", err)
		}

		imagePath := ""
		if source := resolveImageSource(entry); source != "" {
			imagePath = filepath.Join(thumbsDir, slug+"-img.png")
			// Regenerate each run so wallpaper/source changes are reflected.
			_ = os.Remove(imagePath)
			jobs = append(jobs, thumbJob{name: entry.theme.Name, source: source, dst: imagePath})
		}

		man.Themes = append(man.Themes, manifestTheme{
			Name:      entry.theme.Name,
			Path:      entry.path,
			Thumbnail: palettePath,
			Image:     imagePath,
			Font:      entry.theme.Font.Family,
			Colors:    paletteMap(entry.theme.Colors),
		})
	}

	return man, jobs
}

// generateThumbnails renders the deferred real preview images while the picker is
// already on screen. It stops early if the context is cancelled (picker closed).
func generateThumbnails(ctx context.Context, jobs []thumbJob) {
	var wg sync.WaitGroup
	// A small worker pool keeps the UI responsive without spawning an unbounded
	// number of ffmpeg processes at once.
	const workers = 3
	jobCh := make(chan thumbJob)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				if ctx.Err() != nil {
					return
				}
				if err := renderImageThumbnail(ctx, job.source, job.dst); err != nil {
					selectorLogger.Debug("Failed to render preview image", "theme", job.name, "error", err)
				}
			}
		}()
	}

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			close(jobCh)
			wg.Wait()
			return
		case jobCh <- job:
		}
	}
	close(jobCh)
	wg.Wait()
}

// resolveImageSource returns an absolute path to the best available image or
// video source for a real preview, or an empty string when none exists. Lottie
// and shader sources are only used when a previously rendered MP4 exists in the
// shared wallpaper cache; rendering them from scratch would be too slow here.
func resolveImageSource(entry themeEntry) string {
	if entry.theme.Thumbnail != "" {
		if resolved := resolveRelative(entry.dir, entry.theme.Thumbnail); resolved != "" {
			return resolved
		}
	}

	for _, pattern := range entry.theme.Wallpapers.Images {
		if resolved := resolveRelative(entry.dir, pattern); resolved != "" {
			return resolved
		}
	}

	for _, pattern := range entry.theme.Wallpapers.Videos {
		if resolved := resolveRelative(entry.dir, pattern); resolved != "" {
			return resolved
		}
	}

	for _, pattern := range entry.theme.Wallpapers.Lotties {
		if cached := cachedRenderForStem(lottieCacheDir(), pattern); cached != "" {
			return cached
		}
	}

	for _, shader := range entry.theme.Wallpapers.Shaders {
		if cached := cachedRenderForStem(glslCacheDir(), shader.Path); cached != "" {
			return cached
		}
	}

	return ""
}

// resolveRelative resolves a path or glob relative to baseDir to a single
// existing file, or returns an empty string.
func resolveRelative(baseDir string, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	if matches, _ := filepath.Glob(path); len(matches) > 0 {
		sort.Strings(matches)
		return matches[0]
	}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}
	return ""
}

// cachedRenderForStem looks for a previously rendered MP4 for a lottie/shader
// source in the shared wallpaper cache, returning the newest match.
func cachedRenderForStem(cacheDir string, sourcePath string) string {
	stem := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	if stem == "" {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(cacheDir, stem+"-*.mp4"))
	if len(matches) == 0 {
		return ""
	}

	newest := matches[0]
	var newestMod int64
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if mod := info.ModTime().UnixNano(); mod >= newestMod {
			newestMod = mod
			newest = match
		}
	}
	return newest
}

func lottieCacheDir() string {
	return filepath.Join(cacheBaseDir(), "shell-dev", "lottie-cache")
}

func glslCacheDir() string {
	return filepath.Join(cacheBaseDir(), "shell-dev", "glsl-cache")
}

func cacheBaseDir() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return xdg
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache")
}

// renderImageThumbnail uses ffmpeg to pick a representative frame and cover-crop
// it to the thumbnail dimensions. It handles both still images and video sources
// (the thumbnail filter selects a frame). ffmpeg is a hard dependency of Rainbeau.
func renderImageThumbnail(ctx context.Context, src string, dst string) error {
	filter := fmt.Sprintf(
		"thumbnail,scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d",
		thumbWidth, thumbHeight, thumbWidth, thumbHeight,
	)
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-loglevel", "error", "-i", src, "-vf", filter, "-frames:v", "1", dst)
	return cmd.Run()
}

// renderPaletteThumbnail draws the theme palette as vertical bands covering every
// color in the palette, used as the fallback thumbnail when no image is available.
func renderPaletteThumbnail(colors rainbeau.ThemeColors, dst string) error {
	bands := []string{
		colors.Bg0, colors.Bg1, colors.Bg2, colors.Bg3,
		colors.Border,
		colors.Accent1, colors.Accent2,
		colors.Text, colors.TextDim,
		colors.Red, colors.Green, colors.Blue,
		colors.Inactive,
	}

	img := image.NewRGBA(image.Rect(0, 0, thumbWidth, thumbHeight))
	for i, hex := range bands {
		startX := i * thumbWidth / len(bands)
		endX := (i + 1) * thumbWidth / len(bands)
		draw.Draw(img, image.Rect(startX, 0, endX, thumbHeight), &image.Uniform{C: parseHexColor(hex)}, image.Point{}, draw.Src)
	}

	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func paletteMap(colors rainbeau.ThemeColors) map[string]string {
	return map[string]string{
		"bg0":      colors.Bg0,
		"bg1":      colors.Bg1,
		"bg2":      colors.Bg2,
		"bg3":      colors.Bg3,
		"border":   colors.Border,
		"accent1":  colors.Accent1,
		"accent2":  colors.Accent2,
		"text":     colors.Text,
		"text_dim": colors.TextDim,
		"red":      colors.Red,
		"green":    colors.Green,
		"blue":     colors.Blue,
		"inactive": colors.Inactive,
	}
}

func parseHexColor(hex string) color.RGBA {
	clean := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(clean) != 6 {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	var r, g, b int
	_, err := fmt.Sscanf(clean, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func writeManifest(man manifest, path string) error {
	data, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// extractQML writes the embedded QML files to dstDir so quickshell can load them.
func extractQML(dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	return fs.WalkDir(qmlFS, "qml", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := qmlFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(filepath.Join(dstDir, filepath.Base(path)), data, 0o644)
	})
}

func ensureCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "rainbeau")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func slugify(themePath string, index int) string {
	base := filepath.Base(themePath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var builder strings.Builder
	for _, r := range strings.ToLower(base) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	cleaned := strings.Trim(builder.String(), "-")
	if cleaned == "" {
		return fmt.Sprintf("theme-%d", index)
	}
	return cleaned
}
