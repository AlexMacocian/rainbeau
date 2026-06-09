package internal

import (
	"math"
	"strings"
)

type ResolvedPalette struct {
	Bg0           string
	Bg1           string
	Bg2           string
	Bg3           string
	Border        string
	Accent1       string
	Accent2       string
	Text          string
	TextDim       string
	Inactive      string
	Red           string
	Green         string
	Blue          string
	AnsiBlack     string
	AnsiRed       string
	AnsiGreen     string
	AnsiYellow    string
	AnsiBlue      string
	AnsiMagenta   string
	AnsiCyan      string
	AnsiWhite     string
	BrightBlack   string
	BrightRed     string
	BrightGreen   string
	BrightYellow  string
	BrightBlue    string
	BrightMagenta string
	BrightCyan    string
	BrightWhite   string
	SelectionBg   string
	SelectionFg   string
	IsLight       bool
	MinContrast   float64
}

func ResolvePalette(theme *Theme) ResolvedPalette {
	c := theme.Colors
	bg0 := c.Bg0
	isLight := strings.Contains(strings.ToLower(theme.Gtk.ColorScheme), "light")
	minContrast := theme.Terminal.EffectiveMinContrast()
	minUIContrast := math.Min(4.5, minContrast)

	text := EnsureContrast(c.Text, bg0, minContrast)
	textDim := EnsureContrast(c.TextDim, bg0, minUIContrast)
	border := EnsureContrast(c.Border, bg0, minUIContrast)
	accent1 := EnsureContrast(c.Accent1, bg0, minContrast)
	accent2 := EnsureContrast(c.Accent2, bg0, minContrast)
	red := EnsureContrast(c.Red, bg0, minContrast)
	green := EnsureContrast(c.Green, bg0, minContrast)
	blue := EnsureContrast(c.Blue, bg0, minContrast)
	inactive := EnsureContrast(c.Inactive, bg0, 3.0)

	sat := c.SaturationBoost()
	redAnchor := forceMinSaturation(c.Red, 0.55)
	greenAnchor := forceMinSaturation(c.Green, 0.45)
	blueAnchor := forceMinSaturation(c.Blue, 0.50)

	yellowSeed := hueAt(redAnchor, 50, 0.65)
	magentaSeed := hueAt(redAnchor, 320, 0.55)
	cyanSeed := hueAt(blueAnchor, 180, 0.55)

	ansiRed := EnsureContrast(AdjustSaturation(redAnchor, sat), bg0, minContrast)
	ansiGreen := EnsureContrast(AdjustSaturation(greenAnchor, sat), bg0, minContrast)
	ansiYellow := EnsureContrast(yellowSeed, bg0, minContrast)
	ansiBlue := EnsureContrast(AdjustSaturation(blueAnchor, sat), bg0, minContrast)
	ansiMagenta := EnsureContrast(magentaSeed, bg0, minContrast)
	ansiCyan := EnsureContrast(cyanSeed, bg0, minContrast)

	distinct := EnsureDistinct([]string{ansiRed, ansiGreen, ansiYellow, ansiBlue, ansiMagenta, ansiCyan}, bg0, 25, 0.10)
	ansiRed = EnsureContrast(distinct[0], bg0, minContrast)
	ansiGreen = EnsureContrast(distinct[1], bg0, minContrast)
	ansiYellow = EnsureContrast(distinct[2], bg0, minContrast)
	ansiBlue = EnsureContrast(distinct[3], bg0, minContrast)
	ansiMagenta = EnsureContrast(distinct[4], bg0, minContrast)
	ansiCyan = EnsureContrast(distinct[5], bg0, minContrast)

	ansiBlack := c.Bg3
	ansiWhite := text
	if isLight {
		ansiBlack = text
		ansiWhite = c.Bg3
	}

	const brightShift = 0.12
	brighten := func(color string) string {
		if isLight {
			return EnsureContrast(Darken(color, brightShift), bg0, minContrast)
		}
		return EnsureContrast(Lighten(color, brightShift), bg0, minContrast)
	}

	brightBlack := EnsureContrast(Lighten(inactive, 0.15), bg0, 4.5)
	brightWhite := EnsureContrast(Lighten(textDim, 0.10), bg0, minContrast)
	if isLight {
		brightBlack = EnsureContrast(Darken(inactive, 0.10), bg0, 3.0)
		brightWhite = c.Bg2
	}

	selectionBg, selectionFg := resolveSelection(c.Border, bg0, text, isLight)

	return ResolvedPalette{
		Bg0:           bg0,
		Bg1:           c.Bg1,
		Bg2:           c.Bg2,
		Bg3:           c.Bg3,
		Border:        border,
		Accent1:       accent1,
		Accent2:       accent2,
		Text:          text,
		TextDim:       textDim,
		Inactive:      inactive,
		Red:           red,
		Green:         green,
		Blue:          blue,
		AnsiBlack:     ansiBlack,
		AnsiRed:       ansiRed,
		AnsiGreen:     ansiGreen,
		AnsiYellow:    ansiYellow,
		AnsiBlue:      ansiBlue,
		AnsiMagenta:   ansiMagenta,
		AnsiCyan:      ansiCyan,
		AnsiWhite:     ansiWhite,
		BrightBlack:   brightBlack,
		BrightRed:     brighten(ansiRed),
		BrightGreen:   brighten(ansiGreen),
		BrightYellow:  brighten(ansiYellow),
		BrightBlue:    brighten(ansiBlue),
		BrightMagenta: brighten(ansiMagenta),
		BrightCyan:    brighten(ansiCyan),
		BrightWhite:   brightWhite,
		SelectionBg:   selectionBg,
		SelectionFg:   selectionFg,
		IsLight:       isLight,
		MinContrast:   minContrast,
	}
}

func resolveSelection(seed string, bg string, text string, isLight bool) (string, string) {
	h, s, _ := HexToHsl(seed)
	s = math.Max(s, 0.55)

	targetL := 0.72
	if isLight {
		targetL = 0.30
	}
	selBg := EnsureContrast(HslToHex(h, s, targetL), bg, 4.5)

	contrastWithText := ContrastRatio(text, selBg)
	contrastWithBg := ContrastRatio(bg, selBg)
	selFg := bg
	if contrastWithText >= contrastWithBg {
		selFg = text
	}

	return selBg, EnsureContrast(selFg, selBg, 7.0)
}

func hueAt(seed string, targetHue float64, minSaturation float64) string {
	_, s, l := HexToHsl(seed)
	s = math.Max(s, minSaturation)
	return HslToHex(targetHue, s, l)
}

func forceMinSaturation(hex string, minSaturation float64) string {
	h, s, l := HexToHsl(hex)
	if s >= minSaturation {
		return hex
	}
	return HslToHex(h, minSaturation, l)
}
