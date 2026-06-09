package internal

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ToHyprRgba converts "#RRGGBB" to "RRGGBBaa" for Hyprland rgba() format.
func ToHyprRgba(hex string, alpha ...string) string {
	resolvedAlpha := "ee"
	if len(alpha) > 0 {
		resolvedAlpha = alpha[0]
	}

	return strings.TrimPrefix(hex, "#") + resolvedAlpha
}

// ToCSSRgba converts "#RRGGBB" to "rgba(R, G, B, opacity)" for CSS.
func ToCSSRgba(hex string, opacity float64) string {
	r, g, b := mustHexToRgb(hex)
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", r, g, b, formatFloat(opacity))
}

// MixColors mixes two "#RRGGBB" colors. Amount 0.0 = all color1, 1.0 = all color2.
func MixColors(hex1 string, hex2 string, amount float64) string {
	r1, g1, b1 := mustHexToRgb(hex1)
	r2, g2, b2 := mustHexToRgb(hex2)

	r := int(float64(r1)*(1-amount) + float64(r2)*amount)
	g := int(float64(g1)*(1-amount) + float64(g2)*amount)
	b := int(float64(b1)*(1-amount) + float64(b2)*amount)

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// Lighten lightens a color by mixing with white. Amount 0.0 = original, 1.0 = white.
func Lighten(hex string, amount float64) string {
	return MixColors(hex, "#FFFFFF", amount)
}

// Darken darkens a color by mixing with black. Amount 0.0 = original, 1.0 = black.
func Darken(hex string, amount float64) string {
	return MixColors(hex, "#000000", amount)
}

// ShiftHue shifts the hue of a "#RRGGBB" color by the given degrees.
func ShiftHue(hex string, degrees float64) string {
	h, s, l := HexToHsl(hex)
	h = math.Mod(math.Mod(h+degrees, 360)+360, 360)
	return HslToHex(h, s, l)
}

// AdjustSaturation adjusts saturation of a "#RRGGBB" color. Positive increases, negative decreases.
func AdjustSaturation(hex string, amount float64) string {
	h, s, l := HexToHsl(hex)
	s = clamp(s+amount, 0, 1)
	return HslToHex(h, s, l)
}

// HexToHsl converts "#RRGGBB" to HSL (H: 0-360, S: 0-1, L: 0-1).
func HexToHsl(hex string) (float64, float64, float64) {
	ri, gi, bi := mustHexToRgb(hex)
	r := float64(ri) / 255.0
	g := float64(gi) / 255.0
	b := float64(bi) / 255.0

	maxValue := math.Max(r, math.Max(g, b))
	minValue := math.Min(r, math.Min(g, b))
	l := (maxValue + minValue) / 2.0

	if maxValue == minValue {
		return 0, 0, l
	}

	d := maxValue - minValue
	s := d / (maxValue + minValue)
	if l > 0.5 {
		s = d / (2.0 - maxValue - minValue)
	}

	var h float64
	switch maxValue {
	case r:
		offset := 0.0
		if g < b {
			offset = 6.0
		}
		h = ((g-b)/d + offset) * 60
	case g:
		h = ((b-r)/d + 2) * 60
	default:
		h = ((r-g)/d + 4) * 60
	}

	return h, s, l
}

// HslToHex converts HSL (H: 0-360, S: 0-1, L: 0-1) to "#RRGGBB".
func HslToHex(h float64, s float64, l float64) string {
	if s == 0 {
		v := int(math.Round(l * 255))
		return fmt.Sprintf("#%02X%02X%02X", v, v, v)
	}

	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q

	r := hueToRgb(p, q, h/360.0+1.0/3.0)
	g := hueToRgb(p, q, h/360.0)
	b := hueToRgb(p, q, h/360.0-1.0/3.0)

	return fmt.Sprintf("#%02X%02X%02X", int(math.Round(r*255)), int(math.Round(g*255)), int(math.Round(b*255)))
}

// RelativeLuminance computes WCAG 2.x relative luminance for a "#RRGGBB" color.
func RelativeLuminance(hex string) float64 {
	ri, gi, bi := mustHexToRgb(hex)
	rs := float64(ri) / 255.0
	gs := float64(gi) / 255.0
	bs := float64(bi) / 255.0

	r := linearizeSRGB(rs)
	g := linearizeSRGB(gs)
	b := linearizeSRGB(bs)

	return 0.2126*r + 0.7152*g + 0.0722*b
}

// ContrastRatio computes the WCAG contrast ratio between two "#RRGGBB" colors.
func ContrastRatio(hex1 string, hex2 string) float64 {
	l1 := RelativeLuminance(hex1)
	l2 := RelativeLuminance(hex2)
	lighter := math.Max(l1, l2)
	darker := math.Min(l1, l2)
	return (lighter + 0.05) / (darker + 0.05)
}

// EnsureContrast adjusts a foreground color until it meets the minimum WCAG contrast ratio.
func EnsureContrast(fg string, bg string, minRatio ...float64) string {
	resolvedMinRatio := 4.5
	if len(minRatio) > 0 {
		resolvedMinRatio = minRatio[0]
	}

	if ContrastRatio(fg, bg) >= resolvedMinRatio {
		return fg
	}

	bgLum := RelativeLuminance(bg)
	fgLum := RelativeLuminance(fg)
	h, s, l := HexToHsl(fg)

	step := -0.02
	if fgLum >= bgLum {
		step = 0.02
	}
	if math.Abs(fgLum-bgLum) < 0.01 {
		step = 0.02
		if bgLum > 0.5 {
			step = -0.02
		}
	}

	for range 80 {
		l = clamp(l+step, 0.02, 0.98)
		candidate := HslToHex(h, s, l)
		if ContrastRatio(candidate, bg) >= resolvedMinRatio {
			return candidate
		}
	}

	for range 40 {
		s = math.Max(0, s-0.04)
		l = clamp(l+step, 0.02, 0.98)
		candidate := HslToHex(h, s, l)
		if ContrastRatio(candidate, bg) >= resolvedMinRatio {
			return candidate
		}
	}

	return HslToHex(h, s, l)
}

// EnsureDistinct nudges foreground colors apart when their hue and lightness are too similar.
func EnsureDistinct(colors []string, bg string, options ...float64) []string {
	minHueDelta := 25.0
	minLightnessDelta := 0.08
	if len(options) > 0 {
		minHueDelta = options[0]
	}
	if len(options) > 1 {
		minLightnessDelta = options[1]
	}

	type hsl struct {
		H float64
		S float64
		L float64
	}

	result := make([]hsl, len(colors))
	for i, color := range colors {
		h, s, l := HexToHsl(color)
		result[i] = hsl{H: h, S: s, L: l}
	}

	for range 3 {
		for i := range result {
			for j := i + 1; j < len(result); j++ {
				hueDiff := math.Abs(result[i].H - result[j].H)
				if hueDiff > 180 {
					hueDiff = 360 - hueDiff
				}

				lDiff := math.Abs(result[i].L - result[j].L)
				if hueDiff < minHueDelta && lDiff < minLightnessDelta {
					bgLum := RelativeLuminance(bg)
					if bgLum > 0.5 {
						result[i].L = clamp(result[i].L-minLightnessDelta/2, 0.10, 0.85)
						result[j].L = clamp(result[j].L+minLightnessDelta/2, 0.10, 0.85)
					} else {
						result[i].L = clamp(result[i].L+minLightnessDelta/2, 0.15, 0.90)
						result[j].L = clamp(result[j].L-minLightnessDelta/2, 0.15, 0.90)
					}
				}
			}
		}
	}

	output := make([]string, len(colors))
	for i, value := range result {
		output[i] = HslToHex(value.H, value.S, value.L)
	}

	return output
}

func hueToRgb(p float64, q float64, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func linearizeSRGB(value float64) float64 {
	if value <= 0.03928 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

func mustHexToRgb(hex string) (int, int, int) {
	clean := strings.TrimPrefix(hex, "#")
	r, err := strconv.ParseInt(clean[0:2], 16, 64)
	if err != nil {
		panic(err)
	}
	g, err := strconv.ParseInt(clean[2:4], 16, 64)
	if err != nil {
		panic(err)
	}
	b, err := strconv.ParseInt(clean[4:6], 16, 64)
	if err != nil {
		panic(err)
	}
	return int(r), int(g), int(b)
}

func clamp(value float64, minValue float64, maxValue float64) float64 {
	return math.Min(math.Max(value, minValue), maxValue)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
