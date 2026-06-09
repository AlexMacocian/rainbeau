package main

type Theme struct {
	Name       string            `json:"name"`
	Colors     ThemeColors       `json:"colors"`
	Hyprland   HyprlandSettings  `json:"hyprland"`
	Font       FontSettings      `json:"font"`
	Gtk        GtkSettings       `json:"gtk"`
	Waybar     WaybarSettings    `json:"waybar"`
	Wallpapers WallpaperSettings `json:"wallpapers"`
	Terminal   TerminalSettings  `json:"terminal"`
	Nvim       NvimSettings      `json:"nvim"`
}

// ThemeColors describes the color palette for the theme, including background colors, border color, accent colors, text colors, and saturation.
type ThemeColors struct {
	Bg0        string  `json:"bg0"`
	Bg1        string  `json:"bg1"`
	Bg2        string  `json:"bg2"`
	Bg3        string  `json:"bg3"`
	Border     string  `json:"border"`
	Accent1    string  `json:"accent1"`
	Accent2    string  `json:"accent2"`
	Text       string  `json:"text"`
	TextDim    string  `json:"text_dim"`
	Red        string  `json:"red"`
	Green      string  `json:"green"`
	Blue       string  `json:"blue"`
	Inactive   string  `json:"inactive"`
	Saturation float64 `json:"saturation"`
}

// HyprlandSettings describes the settings for the Hyprland configuration, such as border size, rounding, gaps, shadow, blur, opacity, and animation speeds.
type HyprlandSettings struct {
	BorderSize               int     `json:"border_size"`
	Rounding                 int     `json:"rounding"`
	GapsIn                   int     `json:"gaps_in"`
	GapsOut                  int     `json:"gaps_out"`
	ShadowRange              int     `json:"shadow_range"`
	ShadowRenderPower        int     `json:"shadow_render_power"`
	BlurSize                 int     `json:"blur_size"`
	BlurPasses               int     `json:"blur_passes"`
	BlurVibrancy             float64 `json:"blur_vibrancy"`
	ActiveOpacity            float64 `json:"active_opacity"`
	InactiveOpacity          float64 `json:"inactive_opacity"`
	AnimationSpeedGlobal     float64 `json:"animation_speed_global"`
	AnimationSpeedBorder     float64 `json:"animation_speed_border"`
	AnimationSpeedWindows    float64 `json:"animation_speed_windows"`
	AnimationSpeedWindowsIn  float64 `json:"animation_speed_windows_in"`
	AnimationSpeedWindowsOut float64 `json:"animation_speed_windows_out"`
	AnimationSpeedFadeIn     float64 `json:"animation_speed_fade_in"`
	AnimationSpeedFadeOut    float64 `json:"animation_speed_fade_out"`
	AnimationSpeedWorkspaces float64 `json:"animation_speed_workspaces"`
}

// FontSettings describes the font family, fallback fonts, and size to be used in the theme.
type FontSettings struct {
	Family   string   `json:"family"`
	Fallback []string `json:"fallback"`
	Size     int      `json:"size"`
}

// WaybarSettings describes the settings for the waybar configuration, such as height, separator, opacity, border width, and workspace labels.
type WaybarSettings struct {
	Height          int      `json:"height"`
	Separator       string   `json:"separator"`
	Opacity         float64  `json:"opacity"`
	BorderWidth     int      `json:"border_width"`
	WorkspaceLabels []string `json:"workspace_labels"`
}

// WallpaperSettings describes the wallpapers to be used and how they should be rendered.
type WallpaperSettings struct {
	FitMode       string        `json:"fit_mode"`
	Images        []string      `json:"images"`
	Videos        []string      `json:"videos"`
	CycleInterval int           `json:"cycle_interval"`
	Lotties       []string      `json:"lotties"`
	Shaders       []ShaderEntry `json:"shaders"`
}

// ShaderEntry describes a single shader to be rendered as a wallpaper.
// The path has to direct to a fragment shader file.
type ShaderEntry struct {
	Path            string `json:"path"`
	DurationSeconds int    `json:"duration_seconds"`
	Fps             int    `json:"fps"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
}

// GtkSettings describes the per-theme GTK color scheme and theme selection.
type GtkSettings struct {
	ColorScheme string `json:"color_scheme"`
	Theme       string `json:"theme"`
}

// TerminalSettings describes terminal specific tuning. All fields
// are optional and sensible defaults will be used when omitted.
type TerminalSettings struct {
	Opacity     float64 `json:"opacity"`
	MinContrast float64 `json:"min_contrast"`
}

// NvimSettings describes the per-theme Neovim colorscheme selection.
//
// Each theme picks a real, hand-tuned nvim colorscheme rather than trying to
// dynamically rewrite catppuccin's palette. Dynamic palette overrides interact
// poorly with treesitter/LSP highlight groups and looked off in practice.
//
// Colorscheme is the name passed to :colorscheme, for example
// "catppuccin-mocha", "slate", or "modus_vivendi".
//
// Plugin is an optional lazy.nvim spec in owner/repo format. Omit it for
// nvim's built-in colorschemes such as slate, koehler, peachpuff,
// modus_operandi, etc.
//
// Name is an optional override for lazy.nvim's name field, for example
// "catppuccin" for "catppuccin/nvim".
type NvimSettings struct {
	ColorScheme string `json:"colorscheme"`
	Plugin      string `json:"plugin"`
	Name        string `json:"name"`
}

func (t TerminalSettings) EffectiveOpacity() float64 {
	if t.Opacity == 0 {
		return 1
	}

	return t.Opacity
}

func (t TerminalSettings) EffectiveMinContrast() float64 {
	if t.MinContrast == 0 {
		return 7.0
	}

	return t.MinContrast
}
