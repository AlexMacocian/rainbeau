package main

/*
#cgo linux LDFLAGS: -lrlottie
#include <stdint.h>
#include <stdlib.h>

typedef struct Lottie_Animation_S Lottie_Animation;

Lottie_Animation* lottie_animation_from_file(const char* path);
void lottie_animation_destroy(Lottie_Animation* animation);
void lottie_animation_get_size(Lottie_Animation* animation, size_t* width, size_t* height);
size_t lottie_animation_get_totalframe(Lottie_Animation* animation);
double lottie_animation_get_framerate(Lottie_Animation* animation);
void lottie_animation_render(
	Lottie_Animation* animation,
	size_t frame_num,
	uint32_t* buffer,
	size_t width,
	size_t height,
	size_t bytes_per_line);
*/
import "C"

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var lottieLogger = getLogger("lottie", slog.LevelInfo)

// convertLotties renders Lottie sources under wallpapersDir into cached MP4 files.
func convertLotties(lottiePaths []string, wallpapersDir string, bgHex string, lineHex string) []string {
	if len(lottiePaths) == 0 {
		return nil
	}

	bg := normalizeHex(bgHex)
	line := normalizeHex(lineHex)

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		lottieLogger.Warn("ffmpeg not found on PATH; install it to enable Lottie wallpapers", "error", err)
		return nil
	}

	rlottie := rlottieLibrary{}

	cacheDir, err := lottieCacheDir()
	if err != nil {
		lottieLogger.Warn("failed to resolve Lottie cache directory", "error", err)
		return nil
	}

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		lottieLogger.Warn("failed to create Lottie cache directory", "path", cacheDir, "error", err)
		return nil
	}

	var outputs []string
	for _, lottiePath := range lottiePaths {
		sourcePath := lottiePath
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(wallpapersDir, sourcePath)
		}

		if _, err := os.Stat(sourcePath); err != nil {
			lottieLogger.Warn("Lottie source not found", "path", lottiePath, "error", err)
			continue
		}

		stem := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
		mp4Abs := filepath.Join(cacheDir, fmt.Sprintf("%s-%s-%s.mp4", stem, bg, line))

		if _, err := os.Stat(mp4Abs); err == nil {
			lottieLogger.Info("Lottie cache hit", "source", lottiePath, "output", mp4Abs)
			outputs = append(outputs, mp4Abs)
			continue
		}

		lottieLogger.Info("Rendering Lottie", "source", lottiePath, "output", mp4Abs)

		sourceJSONPath := sourcePath
		extracted := ""
		if strings.EqualFold(filepath.Ext(sourcePath), ".lottie") {
			extracted, err = extractDotLottieJSON(sourcePath)
			if err != nil {
				lottieLogger.Warn("could not extract animation JSON from dotLottie file", "path", lottiePath, "error", err)
				continue
			}
			sourceJSONPath = extracted
		}

		prepared := prepareLottieJSON(sourceJSONPath, line, bg)

		if !runLottieConvert(&rlottie, prepared, mp4Abs) {
			if err := os.Remove(mp4Abs); err != nil && !errors.Is(err, os.ErrNotExist) {
				lottieLogger.Warn("failed to remove partial Lottie output", "path", mp4Abs, "error", err)
			}
			lottieLogger.Warn("failed to render Lottie", "path", lottiePath)
			cleanupLottieTempFiles(extracted, prepared, sourceJSONPath)
			continue
		}

		cleanupLottieTempFiles(extracted, prepared, sourceJSONPath)
		outputs = append(outputs, mp4Abs)
	}

	return outputs
}

func lottieCacheDir() (string, error) {
	xdg := os.Getenv("XDG_CACHE_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg = filepath.Join(home, ".cache")
	}

	return filepath.Join(xdg, "shell-dev", "lottie-cache"), nil
}

func extractDotLottieJSON(dotLottiePath string) (string, error) {
	reader, err := zip.OpenReader(dotLottiePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			lottieLogger.Warn("failed to close dotLottie archive", "path", dotLottiePath, "error", err)
		}
	}()

	for _, file := range reader.File {
		if !strings.HasPrefix(strings.ToLower(file.Name), "animations/") || !strings.HasSuffix(strings.ToLower(file.Name), ".json") {
			continue
		}

		src, err := file.Open()
		if err != nil {
			return "", err
		}

		dst, err := os.CreateTemp("", "lottie-*.json")
		if err != nil {
			if closeErr := src.Close(); closeErr != nil {
				lottieLogger.Warn("failed to close dotLottie entry", "path", file.Name, "error", closeErr)
			}
			return "", err
		}

		if _, err := io.Copy(dst, src); err != nil {
			if closeErr := src.Close(); closeErr != nil {
				lottieLogger.Warn("failed to close dotLottie entry", "path", file.Name, "error", closeErr)
			}
			if closeErr := dst.Close(); closeErr != nil {
				lottieLogger.Warn("failed to close temp Lottie JSON after copy error", "path", dst.Name(), "error", closeErr)
			}
			removeLottieTempFile(dst.Name(), "failed to remove temp Lottie JSON after copy error")
			return "", err
		}

		if err := src.Close(); err != nil {
			if closeErr := dst.Close(); closeErr != nil {
				lottieLogger.Warn("failed to close temp Lottie JSON after source close error", "path", dst.Name(), "error", closeErr)
			}
			removeLottieTempFile(dst.Name(), "failed to remove temp Lottie JSON after source close error")
			return "", err
		}

		if err := dst.Close(); err != nil {
			removeLottieTempFile(dst.Name(), "failed to remove temp Lottie JSON after close error")
			return "", err
		}

		return dst.Name(), nil
	}

	return "", errors.New("no animations/*.json entry found")
}

func prepareLottieJSON(jsonPath string, lineHex string, bgHex string) string {
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		return jsonPath
	}

	r, g, b, err := hexToFloatRGB(lineHex)
	if err != nil {
		return jsonPath
	}

	prepared, err := recolorLottie(content, r, g, b)
	if err != nil {
		return jsonPath
	}

	prepared, err = injectBackgroundLayer(prepared, bgHex)
	if err != nil {
		return jsonPath
	}

	out, err := os.CreateTemp("", "lottie-prep-*.json")
	if err != nil {
		return jsonPath
	}

	if _, err := out.Write(prepared); err != nil {
		if closeErr := out.Close(); closeErr != nil {
			lottieLogger.Warn("failed to close prepared Lottie JSON after write error", "path", out.Name(), "error", closeErr)
		}
		removeLottieTempFile(out.Name(), "failed to remove prepared Lottie JSON after write error")
		return jsonPath
	}

	if err := out.Close(); err != nil {
		removeLottieTempFile(out.Name(), "failed to remove prepared Lottie JSON after close error")
		return jsonPath
	}

	return out.Name()
}

func injectBackgroundLayer(content []byte, bgHex string) ([]byte, error) {
	root := gjson.ParseBytes(content)
	if !root.Get("layers").IsArray() {
		return content, nil
	}

	w := root.Get("w")
	h := root.Get("h")
	op := root.Get("op")
	if !w.Exists() || !h.Exists() || !op.Exists() {
		return content, nil
	}

	r, g, b, err := hexToFloatRGB(bgHex)
	if err != nil {
		return content, err
	}

	bgLayer := fmt.Sprintf(
		`{"ddd":0,"ind":9999,"ty":4,"nm":"ThemeEngineBackground","sr":1,"ks":{"o":{"a":0,"k":100},"r":{"a":0,"k":0},"p":{"a":0,"k":[%s,%s,0]},"a":{"a":0,"k":[0,0,0]},"s":{"a":0,"k":[100,100,100]}},"ao":0,"shapes":[{"ty":"gr","it":[{"ty":"rc","d":1,"s":{"a":0,"k":[%s,%s]},"p":{"a":0,"k":[0,0]},"r":{"a":0,"k":0}},{"ty":"fl","c":{"a":0,"k":[%s,%s,%s,1]},"o":{"a":0,"k":100},"r":1,"bm":0},{"ty":"tr","p":{"a":0,"k":[0,0]},"a":{"a":0,"k":[0,0]},"s":{"a":0,"k":[100,100]},"r":{"a":0,"k":0},"o":{"a":0,"k":100},"sk":{"a":0,"k":0},"sa":{"a":0,"k":0}}]}],"ip":0,"op":%s,"st":0,"bm":0}`,
		formatLottieNumber(w.Float()/2.0),
		formatLottieNumber(h.Float()/2.0),
		formatLottieNumber(w.Float()),
		formatLottieNumber(h.Float()),
		formatLottieNumber(r),
		formatLottieNumber(g),
		formatLottieNumber(b),
		formatLottieNumber(op.Float()),
	)

	return sjson.SetRawBytes(content, "layers.-1", []byte(bgLayer))
}

func recolorLottie(content []byte, r float64, g float64, b float64) ([]byte, error) {
	var err error
	walkLottieJSON(gjson.ParseBytes(content), "", func(path string, node gjson.Result) {
		if err != nil {
			return
		}

		ty := node.Get("ty").String()
		colorValues := node.Get("c.k")
		if (ty == "st" || ty == "fl") && colorValues.IsArray() {
			alpha := 1.0
			if parsedAlpha := colorValues.Get("3"); parsedAlpha.Exists() {
				alpha = parsedAlpha.Float()
			}

			colorPath := joinJSONPath(path, "c.k")
			color := fmt.Sprintf("[%s,%s,%s,%s]", formatLottieNumber(r), formatLottieNumber(g), formatLottieNumber(b), formatLottieNumber(alpha))
			content, err = sjson.SetRawBytes(content, colorPath, []byte(color))
		}
	})

	return content, err
}

func walkLottieJSON(node gjson.Result, path string, visit func(string, gjson.Result)) {
	if node.IsObject() {
		visit(path, node)
		node.ForEach(func(key gjson.Result, value gjson.Result) bool {
			walkLottieJSON(value, joinJSONPath(path, escapeJSONPathKey(key.String())), visit)
			return true
		})
		return
	}

	if node.IsArray() {
		index := 0
		node.ForEach(func(_ gjson.Result, value gjson.Result) bool {
			walkLottieJSON(value, joinJSONPath(path, strconv.Itoa(index)), visit)
			index++
			return true
		})
	}
}

func joinJSONPath(base string, child string) string {
	if base == "" {
		return child
	}

	return base + "." + child
}

func escapeJSONPathKey(key string) string {
	key = strings.ReplaceAll(key, `\`, `\\`)
	key = strings.ReplaceAll(key, `.`, `\.`)
	key = strings.ReplaceAll(key, `:`, `\:`)
	return key
}

func formatLottieNumber(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func hexToFloatRGB(hex string) (float64, float64, float64, error) {
	normalized := normalizeHex(hex)
	if len(normalized) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color %q", hex)
	}

	r, err := strconv.ParseInt(normalized[0:2], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	g, err := strconv.ParseInt(normalized[2:4], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	b, err := strconv.ParseInt(normalized[4:6], 16, 64)
	if err != nil {
		return 0, 0, 0, err
	}

	return float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0, nil
}

func normalizeHex(hex string) string {
	normalized := strings.TrimLeft(hex, "#")
	if len(normalized) == 8 {
		normalized = normalized[:6]
	}

	return strings.ToLower(normalized)
}

func runLottieConvert(rlottie *rlottieLibrary, input string, output string) bool {
	animation := rlottie.fromFile(input)
	if animation == nil {
		return false
	}
	defer rlottie.destroy(animation)

	width, height := rlottie.getSize(animation)
	frames := rlottie.getTotalFrame(animation)
	fps := rlottie.getFrameRate(animation)
	if width <= 0 || height <= 0 || frames <= 0 || fps <= 0 {
		return false
	}

	cmd := exec.Command(
		"ffmpeg",
		"-y", "-hide_banner", "-loglevel", "error",
		"-f", "rawvideo",
		"-pix_fmt", "bgra",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-r", strconv.FormatFloat(fps, 'g', -1, 64),
		"-i", "-",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-preset", "veryfast",
		"-crf", "23",
		"-movflags", "+faststart",
		output,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		lottieLogger.Warn("failed to open ffmpeg stdin", "error", err)
		return false
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		lottieLogger.Warn("failed to start ffmpeg", "error", err)
		return false
	}

	bytesPerLine := width * 4
	frameBytes := width * height * 4
	frameBuffer := make([]byte, frameBytes)

	for frame := range frames {
		rlottie.render(animation, frame, frameBuffer, width, height, bytesPerLine)
		if _, err := stdin.Write(frameBuffer); err != nil {
			if closeErr := stdin.Close(); closeErr != nil {
				lottieLogger.Warn("failed to close ffmpeg stdin after write error", "error", closeErr)
			}
			if waitErr := cmd.Wait(); waitErr != nil {
				lottieLogger.Warn("ffmpeg failed after Lottie frame write error", "error", waitErr, "stderr", strings.TrimSpace(stderr.String()))
			}
			lottieLogger.Warn("failed to write Lottie frame to ffmpeg", "frame", frame, "error", err)
			return false
		}
	}

	if err := stdin.Close(); err != nil {
		if waitErr := cmd.Wait(); waitErr != nil {
			lottieLogger.Warn("ffmpeg failed after stdin close error", "error", waitErr, "stderr", strings.TrimSpace(stderr.String()))
		}
		lottieLogger.Warn("failed to close ffmpeg stdin", "error", err)
		return false
	}

	if err := cmd.Wait(); err != nil {
		lottieLogger.Warn("ffmpeg failed", "error", err, "stderr", strings.TrimSpace(stderr.String()))
		return false
	}

	if _, err := os.Stat(output); err != nil {
		lottieLogger.Warn("ffmpeg did not create output file", "path", output, "error", err)
		return false
	}

	return true
}

func cleanupLottieTempFiles(extracted string, prepared string, originalSource string) {
	if extracted != "" {
		removeLottieTempFile(extracted, "failed to remove extracted Lottie temp file")
	}

	if prepared != originalSource && prepared != extracted {
		removeLottieTempFile(prepared, "failed to remove prepared Lottie temp file")
	}
}

func removeLottieTempFile(path string, message string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		lottieLogger.Warn(message, "path", path, "error", err)
	}
}

type rlottieLibrary struct{}

func (r *rlottieLibrary) fromFile(path string) *C.Lottie_Animation {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	return C.lottie_animation_from_file(cPath)
}

func (r *rlottieLibrary) destroy(animation *C.Lottie_Animation) {
	C.lottie_animation_destroy(animation)
}

func (r *rlottieLibrary) getSize(animation *C.Lottie_Animation) (int, int) {
	var width C.size_t
	var height C.size_t

	C.lottie_animation_get_size(animation, &width, &height)

	return int(width), int(height)
}

func (r *rlottieLibrary) getTotalFrame(animation *C.Lottie_Animation) int {
	return int(C.lottie_animation_get_totalframe(animation))
}

func (r *rlottieLibrary) getFrameRate(animation *C.Lottie_Animation) float64 {
	return float64(C.lottie_animation_get_framerate(animation))
}

func (r *rlottieLibrary) render(animation *C.Lottie_Animation, frame int, buffer []byte, width int, height int, bytesPerLine int) {
	C.lottie_animation_render(
		animation,
		C.size_t(frame),
		(*C.uint32_t)(unsafe.Pointer(&buffer[0])),
		C.size_t(width),
		C.size_t(height),
		C.size_t(bytesPerLine),
	)
}
