package generators

import (
	"encoding/json"
	"testing"

	rainbeau "github.com/AlexMacocian/rainbeau/internal"
)

func retrobox() *rainbeau.Theme {
	th := &rainbeau.Theme{Name: "Retrobox"}
	th.Colors = rainbeau.ThemeColors{
		Bg0: "#1c1c1c", Bg1: "#262626", Bg2: "#303030", Bg3: "#3c3836",
		Border: "#fabd2f", Accent1: "#b8bb26", Accent2: "#fe8019",
		Text: "#ebdbb2", TextDim: "#a89984", Red: "#fb5944",
		Green: "#b8bb26", Blue: "#83a598", Inactive: "#504945",
	}
	th.Font = rainbeau.FontSettings{Family: "JetBrainsMono Nerd Font", Size: 12}
	th.Hyprland = rainbeau.HyprlandSettings{Rounding: 4}
	th.Waybar = rainbeau.WaybarSettings{Height: 34, Opacity: 0.82}
	return th
}

func generateOmniShell(t *testing.T, th *rainbeau.Theme) map[string]any {
	t.Helper()
	out, err := OmniShellConfigGenerator{}.Generate(th, "")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	return m
}

// Themes written before the `shell` section existed only declare `waybar`.
// Those must keep theming omni-shell identically.
func TestOmniShellInheritsWaybarSettings(t *testing.T) {
	m := generateOmniShell(t, retrobox())

	if got := m["opacity"]; got != 0.82 {
		t.Errorf("opacity = %v, want the waybar value 0.82", got)
	}
	if got := m["barHeight"]; got != float64(34) {
		t.Errorf("barHeight = %v, want the waybar value 34", got)
	}
	if got := m["radius"]; got != float64(8) {
		t.Errorf("radius = %v, want rounding*2 = 8", got)
	}
	if got := m["opaqueOpacity"].(float64); got <= 0.82 || got > 1.0 {
		t.Errorf("opaqueOpacity = %v, want above the panel opacity and at most 1.0", got)
	}
}

// An explicit `shell` section takes precedence over the waybar fallback.
func TestOmniShellSectionOverridesWaybar(t *testing.T) {
	th := retrobox()
	th.Shell = rainbeau.ShellSettings{Height: 40, Opacity: 0.5, Radius: 20}

	m := generateOmniShell(t, th)

	if got := m["opacity"]; got != 0.5 {
		t.Errorf("opacity = %v, want 0.5", got)
	}
	if got := m["barHeight"]; got != float64(40) {
		t.Errorf("barHeight = %v, want 40", got)
	}
	if got := m["radius"]; got != float64(20) {
		t.Errorf("radius = %v, want 20", got)
	}
}

// The shell renders a three-level text hierarchy. If active is not brighter
// than foreground, focused items read as dimmer than unfocused ones.
func TestOmniShellTextHierarchyIsOrdered(t *testing.T) {
	colors := generateOmniShell(t, retrobox())["colors"].(map[string]any)

	active := rainbeau.RelativeLuminance(colors["active"].(string))
	foreground := rainbeau.RelativeLuminance(colors["foreground"].(string))
	idle := rainbeau.RelativeLuminance(colors["idle"].(string))

	if !(active > foreground && foreground > idle) {
		t.Errorf("want active > foreground > idle, got active=%v foreground=%v idle=%v",
			active, foreground, idle)
	}
}

// A theme with no waybar or shell section at all must still produce a usable
// config rather than zero opacity and a zero-height bar.
func TestOmniShellFallsBackWithoutAnySection(t *testing.T) {
	th := retrobox()
	th.Waybar = rainbeau.WaybarSettings{}

	m := generateOmniShell(t, th)

	if got := m["opacity"].(float64); got <= 0 || got > 1 {
		t.Errorf("opacity = %v, want a usable default", got)
	}
	if got := m["barHeight"].(float64); got <= 0 {
		t.Errorf("barHeight = %v, want a usable default", got)
	}
}
