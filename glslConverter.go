package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const glslViewerTool = "glslViewer"

var glslLogger = getLogger("glsl", slog.LevelInfo)

var glslPlaceholders = []string{
	"${BG_R}",
	"${BG_G}",
	"${BG_B}",
	"${ACCENT_R}",
	"${ACCENT_G}",
	"${ACCENT_B}",
	"${BG}",
	"${ACCENT}",
	"${LOOP_SECONDS}",
}

// convertShaders renders GLSL fragment shaders to cached MP4 files via headless glslViewer.
func convertShaders(shaders []ShaderEntry, wallpapersDir string, bgHex string, accentHex string) []string {
	if len(shaders) == 0 {
		return nil
	}

	if _, err := exec.LookPath(glslViewerTool); err != nil {
		glslLogger.Warn("glslViewer not found on PATH; install it to enable shader wallpapers", "tool", glslViewerTool, "error", err)
		return nil
	}

	bg := normalizeHex(bgHex)
	accent := normalizeHex(accentHex)
	br, bgGreen, bb, err := hexToFloatRGB(bgHex)
	if err != nil {
		glslLogger.Warn("invalid shader background color", "color", bgHex, "error", err)
		return nil
	}

	ar, ag, ab, err := hexToFloatRGB(accentHex)
	if err != nil {
		glslLogger.Warn("invalid shader accent color", "color", accentHex, "error", err)
		return nil
	}

	cacheDir, err := glslCacheDir()
	if err != nil {
		glslLogger.Warn("failed to resolve GLSL cache directory", "error", err)
		return nil
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		glslLogger.Warn("failed to create GLSL cache directory", "path", cacheDir, "error", err)
		return nil
	}

	var outputs []string
	for _, entry := range shaders {
		sourcePath := entry.Path
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(wallpapersDir, sourcePath)
		}

		if _, err := os.Stat(sourcePath); err != nil {
			glslLogger.Warn("shader source not found", "path", entry.Path, "error", err)
			continue
		}

		source, err := os.ReadFile(sourcePath)
		if err != nil {
			glslLogger.Warn("failed to read shader source", "path", entry.Path, "error", err)
			continue
		}

		hasPlaceholders := shaderHasPlaceholders(string(source))
		stem := strings.TrimSuffix(filepath.Base(entry.Path), filepath.Ext(entry.Path))
		cacheName := fmt.Sprintf("%s-%s-%s-%dx%d-%d-%ds.mp4", stem, bg, accent, entry.Width, entry.Height, entry.Fps, entry.DurationSeconds)
		mp4Abs := filepath.Join(cacheDir, cacheName)

		if _, err := os.Stat(mp4Abs); err == nil {
			glslLogger.Info("shader cache hit", "source", entry.Path, "output", mp4Abs)
			outputs = append(outputs, mp4Abs)
			continue
		}

		shaderToRender := sourcePath
		if hasPlaceholders {
			substituted := substituteShaderColors(string(source), br, bgGreen, bb, ar, ag, ab, entry.DurationSeconds)
			shaderToRender = replaceExtension(mp4Abs, ".frag")
			if err := os.WriteFile(shaderToRender, []byte(substituted), 0o644); err != nil {
				glslLogger.Warn("failed to write substituted shader source", "path", shaderToRender, "error", err)
				continue
			}
		}

		glslLogger.Info("Rendering shader", "source", entry.Path, "output", mp4Abs)

		rawPath := mp4Abs + ".raw.mp4"
		removeFileIfExists(rawPath, "failed to remove stale raw shader output")

		if !runGlslViewer(shaderToRender, rawPath, entry) {
			removeFileIfExists(rawPath, "failed to remove failed raw shader output")
			glslLogger.Warn("failed to render shader", "path", entry.Path)
			continue
		}

		if !compressMp4(rawPath, mp4Abs, float64(entry.DurationSeconds), stem) {
			if err := moveFileOverwrite(rawPath, mp4Abs); err != nil {
				glslLogger.Warn("failed to keep raw shader output after compression failure", "raw", rawPath, "output", mp4Abs, "error", err)
				continue
			}
		} else {
			removeFileIfExists(rawPath, "failed to remove compressed raw shader output")
		}

		outputs = append(outputs, mp4Abs)
	}

	return outputs
}

func glslCacheDir() (string, error) {
	xdg := os.Getenv("XDG_CACHE_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg = filepath.Join(home, ".cache")
	}

	return filepath.Join(xdg, "shell-dev", "glsl-cache"), nil
}

func shaderHasPlaceholders(source string) bool {
	for _, placeholder := range glslPlaceholders {
		if strings.Contains(source, placeholder) {
			return true
		}
	}

	return false
}

func substituteShaderColors(source string, br float64, bg float64, bb float64, ar float64, ag float64, ab float64, loopSeconds int) string {
	replacements := []struct {
		token string
		value string
	}{
		{"${BG_R}", formatGLSLFloat(br)},
		{"${BG_G}", formatGLSLFloat(bg)},
		{"${BG_B}", formatGLSLFloat(bb)},
		{"${ACCENT_R}", formatGLSLFloat(ar)},
		{"${ACCENT_G}", formatGLSLFloat(ag)},
		{"${ACCENT_B}", formatGLSLFloat(ab)},
		{"${BG}", fmt.Sprintf("vec3(%s, %s, %s)", formatGLSLFloat(br), formatGLSLFloat(bg), formatGLSLFloat(bb))},
		{"${ACCENT}", fmt.Sprintf("vec3(%s, %s, %s)", formatGLSLFloat(ar), formatGLSLFloat(ag), formatGLSLFloat(ab))},
		{"${LOOP_SECONDS}", formatGLSLFloat(float64(loopSeconds))},
	}

	result := source
	for _, replacement := range replacements {
		result = strings.ReplaceAll(result, replacement.token, replacement.value)
	}

	return result
}

func formatGLSLFloat(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', 6, 64)
	formatted = strings.TrimRight(formatted, "0")
	if strings.HasSuffix(formatted, ".") {
		return formatted + "0"
	}
	if !strings.Contains(formatted, ".") {
		return formatted + ".0"
	}

	return formatted
}

func runGlslViewer(shaderPath string, outputPath string, entry ShaderEntry) bool {
	cmd := exec.Command(
		glslViewerTool,
		"--headless",
		"--noncurses",
		"-w", strconv.Itoa(entry.Width),
		"-h", strconv.Itoa(entry.Height),
		"-r", strconv.Itoa(entry.Fps),
		"-E", fmt.Sprintf("record,%s,0,%d,%d", outputPath, entry.DurationSeconds, entry.Fps),
		"-E", "q",
		shaderPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		glslLogger.Warn("failed to open glslViewer stdout", "error", err)
		return false
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		glslLogger.Warn("failed to open glslViewer stderr", "error", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		glslLogger.Warn("failed to start glslViewer", "error", err)
		return false
	}

	stem := strings.TrimSuffix(filepath.Base(shaderPath), filepath.Ext(shaderPath))
	stderrDone := collectLines(stderr)
	stdoutDone := logGlslViewerProgress(stdout, stem)

	waitErr := cmd.Wait()
	stdoutErr := <-stdoutDone
	stderrResult := <-stderrDone

	if stdoutErr != nil {
		glslLogger.Warn("failed reading glslViewer stdout", "error", stdoutErr)
	}
	if stderrResult.err != nil {
		glslLogger.Warn("failed reading glslViewer stderr", "error", stderrResult.err)
	}

	if waitErr != nil {
		glslLogger.Warn("glslViewer failed", "error", waitErr, "stderr", strings.TrimSpace(stderrResult.output))
		return false
	}

	if _, err := os.Stat(outputPath); err != nil {
		glslLogger.Warn("glslViewer did not create output file", "path", outputPath, "error", err)
		return false
	}

	return true
}

func compressMp4(srcPath string, dstPath string, totalSeconds float64, stem string) bool {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		glslLogger.Warn("ffmpeg not found, keeping raw shader mp4", "error", err)
		return false
	}

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-progress", "pipe:1",
		"-nostats",
		"-i", srcPath,
		"-c:v", "libx264",
		"-preset", "slow",
		"-crf", "28",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-an",
		dstPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		glslLogger.Warn("failed to open ffmpeg stdout", "error", err)
		return false
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		glslLogger.Warn("failed to open ffmpeg stderr", "error", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		glslLogger.Warn("failed to start ffmpeg", "error", err)
		return false
	}

	stderrDone := collectLines(stderr)
	stdoutDone := logFfmpegProgress(stdout, totalSeconds, stem)

	waitErr := cmd.Wait()
	stdoutErr := <-stdoutDone
	stderrResult := <-stderrDone

	if stdoutErr != nil {
		glslLogger.Warn("failed reading ffmpeg progress", "error", stdoutErr)
	}
	if stderrResult.err != nil {
		glslLogger.Warn("failed reading ffmpeg stderr", "error", stderrResult.err)
	}

	if waitErr != nil {
		glslLogger.Warn("ffmpeg failed", "error", waitErr, "stderr", strings.TrimSpace(stderrResult.output))
		return false
	}

	if _, err := os.Stat(dstPath); err != nil {
		glslLogger.Warn("ffmpeg did not create compressed shader output", "path", dstPath, "error", err)
		return false
	}

	return true
}

type commandOutput struct {
	output string
	err    error
}

func collectLines(pipe io.Reader) <-chan commandOutput {
	done := make(chan commandOutput, 1)
	go func() {
		var output bytes.Buffer
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			output.WriteString(scanner.Text())
			output.WriteByte('\n')
		}
		done <- commandOutput{output: output.String(), err: scanner.Err()}
	}()

	return done
}

func logGlslViewerProgress(pipe io.Reader, stem string) <-chan error {
	done := make(chan error, 1)
	go func() {
		progressRe := regexp.MustCompile(`\[\s*[#.\s]+\]\s*(\d+)\s*%`)
		lastPct := -1
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			match := progressRe.FindStringSubmatch(scanner.Text())
			if len(match) != 2 {
				continue
			}
			pct, err := strconv.Atoi(match[1])
			if err != nil || pct <= lastPct {
				continue
			}
			lastPct = pct
			glslLogger.Info("Rendering shader", "shader", stem, "percent", pct)
		}
		done <- scanner.Err()
	}()

	return done
}

func logFfmpegProgress(pipe io.Reader, totalSeconds float64, stem string) <-chan error {
	done := make(chan error, 1)
	go func() {
		lastPct := -1
		totalUs := totalSeconds * 1_000_000.0
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			if totalUs <= 0 {
				continue
			}

			line := scanner.Text()
			key, value, ok := strings.Cut(line, "=")
			if !ok || (key != "out_time_us" && key != "out_time_ms") {
				continue
			}

			outTime, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				continue
			}

			pct := int(min(max(float64(outTime)*100.0/totalUs, 0), 100))
			if pct <= lastPct {
				continue
			}
			lastPct = pct
			glslLogger.Info("Compressing shader", "shader", stem, "percent", pct)
		}
		done <- scanner.Err()
	}()

	return done
}

func removeFileIfExists(path string, message string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		glslLogger.Warn(message, "path", path, "error", err)
	}
}

func moveFileOverwrite(srcPath string, dstPath string) error {
	if err := os.Remove(dstPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Rename(srcPath, dstPath)
}

func replaceExtension(path string, extension string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + extension
}
