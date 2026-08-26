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

// cantha mirrors the real light theme of the same name. Light themes exercise
// the inverted contrast direction, where prominent text is darker rather than
// lighter.
func cantha() *rainbeau.Theme {
	th := &rainbeau.Theme{Name: "Cantha"}
	th.Colors = rainbeau.ThemeColors{
		Bg0: "#F5EFF5", Bg1: "#EBE3ED", Bg2: "#DED5E2", Bg3: "#D0C5D5",
		Border: "#B8267A", Accent1: "#A82888", Accent2: "#1F8050",
		Text: "#0F0A1A", TextDim: "#3D3450", Red: "#B01515",
		Green: "#1F7A45", Blue: "#1F6F88", Inactive: "#7A7088",
	}
	th.Font = rainbeau.FontSettings{Family: "JetBrainsMono Nerd Font", Size: 12}
	th.Gtk = rainbeau.GtkSettings{ColorScheme: "prefer-light", Theme: "Adwaita"}
	th.Hyprland = rainbeau.HyprlandSettings{Rounding: 6}
	th.Waybar = rainbeau.WaybarSettings{Height: 34, Opacity: 0.88}
	th.Shell = rainbeau.ShellSettings{Height: 34, Opacity: 0.88, Radius: 12}
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

// The shell renders a three-level text hierarchy. Prominence is contrast
// against the background, not raw lightness — so this must hold for light
// themes (where prominent means darker) as well as dark ones.
func TestOmniShellTextHierarchyIsOrdered(t *testing.T) {
	for _, tc := range []struct {
		name  string
		theme *rainbeau.Theme
	}{
		{"dark", retrobox()},
		{"light", cantha()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			colors := generateOmniShell(t, tc.theme)["colors"].(map[string]any)
			bg := colors["background"].(string)

			active := rainbeau.ContrastRatio(colors["active"].(string), bg)
			foreground := rainbeau.ContrastRatio(colors["foreground"].(string), bg)
			idle := rainbeau.ContrastRatio(colors["idle"].(string), bg)

			// active may tie with foreground when the text colour is already at
			// the contrast limit, but it must never fall below it.
			if active < foreground {
				t.Errorf("active is less prominent than foreground: active=%.2f foreground=%.2f", active, foreground)
			}
			if foreground <= idle {
				t.Errorf("foreground is not more prominent than idle: foreground=%.2f idle=%.2f", foreground, idle)
			}
		})
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
