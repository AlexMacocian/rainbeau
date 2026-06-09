package generators

import (
	"encoding/json"
	"fmt"

	rainbeau "github.com/AlexMacocian/rainbeau/internal"
)

type VscodeSettingsGenerator struct{}

func (VscodeSettingsGenerator) Name() string       { return "VS Code Settings" }
func (VscodeSettingsGenerator) OutputPath() string { return ".config/Code/User/settings.json" }
func (VscodeSettingsGenerator) Generate(theme *rainbeau.Theme, wallpapersDir string) (string, error) {
	c := theme.Colors
	p := rainbeau.ResolvePalette(theme)
	baseTheme := "Monokai"
	if p.IsLight {
		baseTheme = "Default Light Modern"
	}
	colors := map[string]string{
		"foreground":                                  p.Text,
		"descriptionForeground":                       p.TextDim,
		"disabledForeground":                          p.Inactive,
		"icon.foreground":                             p.TextDim,
		"textLink.foreground":                         p.Blue,
		"textLink.activeForeground":                   p.Accent1,
		"titleBar.activeBackground":                   c.Bg0,
		"titleBar.activeForeground":                   p.Text,
		"titleBar.inactiveBackground":                 c.Bg0,
		"titleBar.inactiveForeground":                 p.Inactive,
		"activityBar.background":                      c.Bg0,
		"activityBar.foreground":                      p.Border,
		"activityBar.inactiveForeground":              p.Inactive,
		"activityBarBadge.background":                 p.Accent2,
		"activityBarBadge.foreground":                 p.Text,
		"sideBar.background":                          c.Bg1,
		"sideBar.foreground":                          p.Text,
		"sideBar.border":                              c.Bg2,
		"sideBarTitle.foreground":                     p.Border,
		"sideBarSectionHeader.background":             c.Bg2,
		"sideBarSectionHeader.foreground":             p.Accent1,
		"editor.background":                           c.Bg0,
		"editor.foreground":                           p.Text,
		"editor.lineHighlightBackground":              c.Bg1 + "80",
		"editor.selectionBackground":                  p.SelectionBg + "80",
		"editor.selectionHighlightBackground":         p.SelectionBg + "40",
		"editor.wordHighlightBackground":              p.Accent2 + "40",
		"editor.findMatchBackground":                  p.SelectionBg + "60",
		"editor.findMatchHighlightBackground":         p.SelectionBg + "30",
		"editorLineNumber.foreground":                 p.Inactive,
		"editorLineNumber.activeForeground":           p.Border,
		"editorGutter.background":                     c.Bg0,
		"editorGroupHeader.tabsBackground":            c.Bg0,
		"tab.activeBackground":                        c.Bg1,
		"tab.activeForeground":                        p.Accent1,
		"tab.inactiveBackground":                      c.Bg0,
		"tab.inactiveForeground":                      p.TextDim,
		"tab.unfocusedActiveForeground":               p.TextDim,
		"tab.unfocusedInactiveForeground":             p.Inactive,
		"tab.border":                                  c.Bg2,
		"tab.activeBorderTop":                         p.Border,
		"statusBar.background":                        c.Bg0,
		"statusBar.foreground":                        p.TextDim,
		"statusBar.border":                            c.Bg2,
		"statusBar.debuggingBackground":               p.Red,
		"statusBar.debuggingForeground":               p.Text,
		"statusBar.noFolderBackground":                c.Bg1,
		"terminal.background":                         c.Bg0,
		"terminal.foreground":                         p.Text,
		"terminal.selectionBackground":                p.SelectionBg,
		"terminal.ansiBlack":                          p.AnsiBlack,
		"terminal.ansiRed":                            p.AnsiRed,
		"terminal.ansiGreen":                          p.AnsiGreen,
		"terminal.ansiYellow":                         p.AnsiYellow,
		"terminal.ansiBlue":                           p.AnsiBlue,
		"terminal.ansiMagenta":                        p.AnsiMagenta,
		"terminal.ansiCyan":                           p.AnsiCyan,
		"terminal.ansiWhite":                          p.AnsiWhite,
		"terminal.ansiBrightBlack":                    p.BrightBlack,
		"terminal.ansiBrightRed":                      p.BrightRed,
		"terminal.ansiBrightGreen":                    p.BrightGreen,
		"terminal.ansiBrightYellow":                   p.BrightYellow,
		"terminal.ansiBrightBlue":                     p.BrightBlue,
		"terminal.ansiBrightMagenta":                  p.BrightMagenta,
		"terminal.ansiBrightCyan":                     p.BrightCyan,
		"terminal.ansiBrightWhite":                    p.BrightWhite,
		"panel.background":                            c.Bg0,
		"panel.border":                                c.Bg2,
		"panelTitle.activeForeground":                 p.Border,
		"panelTitle.inactiveForeground":               p.Inactive,
		"panelTitle.activeBorder":                     p.Border,
		"list.activeSelectionBackground":              p.SelectionBg + "60",
		"list.activeSelectionForeground":              p.SelectionFg,
		"list.inactiveSelectionBackground":            c.Bg2,
		"list.inactiveSelectionForeground":            p.Text,
		"list.inactiveFocusOutline":                   p.Border + "40",
		"list.hoverBackground":                        c.Bg1,
		"list.highlightForeground":                    p.Accent1,
		"tree.indentGuidesStroke":                     c.Bg3,
		"input.background":                            c.Bg1,
		"input.foreground":                            p.Text,
		"input.border":                                p.Inactive,
		"input.placeholderForeground":                 p.Inactive,
		"focusBorder":                                 p.Border,
		"dropdown.background":                         c.Bg1,
		"dropdown.foreground":                         p.Text,
		"dropdown.border":                             p.Inactive,
		"button.background":                           p.Border,
		"button.foreground":                           c.Bg0,
		"button.hoverBackground":                      p.Accent1,
		"scrollbarSlider.background":                  p.Inactive + "40",
		"scrollbarSlider.hoverBackground":             p.Inactive + "80",
		"scrollbarSlider.activeBackground":            p.Border + "60",
		"breadcrumb.foreground":                       p.TextDim,
		"breadcrumb.focusForeground":                  p.Accent1,
		"breadcrumb.activeSelectionForeground":        p.Border,
		"editorWidget.background":                     c.Bg1,
		"editorWidget.foreground":                     p.Text,
		"editorWidget.border":                         p.Inactive,
		"badge.background":                            p.Accent2,
		"badge.foreground":                            c.Bg0,
		"notificationCenterHeader.foreground":         p.Text,
		"notificationCenterHeader.background":         c.Bg1,
		"notifications.foreground":                    p.Text,
		"notifications.background":                    c.Bg1,
		"notifications.border":                        c.Bg2,
		"minimap.background":                          c.Bg0,
		"peekView.border":                             p.Border,
		"peekViewEditor.background":                   c.Bg1,
		"peekViewResult.background":                   c.Bg0,
		"peekViewTitle.background":                    c.Bg1,
		"peekViewTitleLabel.foreground":               p.Accent1,
		"gitDecoration.modifiedResourceForeground":    p.Border,
		"gitDecoration.untrackedResourceForeground":   p.Green,
		"gitDecoration.deletedResourceForeground":     p.Red,
		"gitDecoration.conflictingResourceForeground": p.Accent2,
		"gitDecoration.ignoredResourceForeground":     p.Inactive,
		"gitDecoration.submoduleResourceForeground":   p.Blue,
		"quickInput.background":                       c.Bg1,
		"quickInput.foreground":                       p.Text,
		"quickInputTitle.background":                  c.Bg2,
		"quickInputList.focusBackground":              p.SelectionBg + "60",
		"quickInputList.focusForeground":              p.SelectionFg,
		"keybindingLabel.foreground":                  p.Text,
		"keybindingLabel.background":                  c.Bg2,
		"keybindingLabel.border":                      c.Bg3,
		"settings.headerForeground":                   p.Accent1,
		"settings.modifiedItemIndicator":              p.Border,
		"settings.focusedRowBackground":               c.Bg1,
		"menu.background":                             c.Bg1,
		"menu.foreground":                             p.Text,
		"menu.selectionBackground":                    p.SelectionBg + "60",
		"menu.selectionForeground":                    p.SelectionFg,
		"menu.separatorBackground":                    c.Bg2,
		"menu.border":                                 c.Bg2,
		"menubar.selectionBackground":                 p.SelectionBg + "60",
		"menubar.selectionForeground":                 p.SelectionFg,
		"editorCodeLens.foreground":                   p.TextDim,
		"editorInlayHint.foreground":                  p.TextDim,
		"editorInlayHint.background":                  c.Bg2 + "80",
		"editorInlayHint.typeForeground":              p.Blue,
		"editorInlayHint.parameterForeground":         p.TextDim,
		"editorBracketMatch.background":               p.Border + "20",
		"editorBracketMatch.border":                   p.Border,
		"widget.shadow":                               c.Bg0 + "40",
	}
	settings := map[string]any{
		"github.copilot.nextEditSuggestions.enabled": true,
		"git.enableSmartCommit":                      true,
		"chat.viewSessions.orientation":              "stacked",
		"workbench.colorTheme":                       baseTheme,
		"editor.fontFamily":                          fmt.Sprintf("'%s', monospace", theme.Font.Family),
		"editor.fontSize":                            theme.Font.Size + 2,
		"terminal.integrated.fontFamily":             fmt.Sprintf("'%s'", theme.Font.Family),
		"terminal.integrated.fontSize":               theme.Font.Size + 2,
		"workbench.colorCustomizations":              colors,
	}
	bytes, err := json.MarshalIndent(settings, "", "  ")
	return string(bytes), err
}

type NvimColorschemeGenerator struct{}

func (NvimColorschemeGenerator) Name() string { return "Neovim Colorscheme" }
func (NvimColorschemeGenerator) OutputPath() string {
	return ".config/nvim/lua/plugins/colorscheme.lua"
}

func (NvimColorschemeGenerator) Generate(theme *rainbeau.Theme, wallpapersDir string) (string, error) {
	if theme.Nvim.ColorScheme != "" {
		return generateStaticNvim(theme), nil
	}
	return generateDynamicNvim(theme), nil
}

func generateStaticNvim(theme *rainbeau.Theme) string {
	nvim := theme.Nvim
	pluginSpec := ""
	if nvim.Plugin != "" {
		nameField := ""
		if nvim.Name != "" {
			nameField = fmt.Sprintf("\n    name = %q,", nvim.Name)
		}
		pluginSpec = fmt.Sprintf(`  {
    %q,%s
    lazy = false,
    priority = 1000,
  },

`, nvim.Plugin, nameField)
	}
	return fmt.Sprintf(`-- Auto-generated by Rainbeau — do not edit manually
-- Theme: %s
return {
%s  {
    "LazyVim/LazyVim",
    opts = { colorscheme = %q },
  },
}
`, theme.Name, pluginSpec, nvim.ColorScheme)
}

func generateDynamicNvim(theme *rainbeau.Theme) string {
	c := theme.Colors
	p := rainbeau.ResolvePalette(theme)
	sat := c.SaturationBoost()
	minContrast := p.MinContrast

	syntaxKeyword := rainbeau.AdjustSaturation(rainbeau.ShiftHue(c.Blue, 90), sat+0.05)
	syntaxType := rainbeau.AdjustSaturation(rainbeau.MixColors(c.Blue, c.Green, 0.60), sat+0.03)
	syntaxMethod := rainbeau.AdjustSaturation(rainbeau.ShiftHue(c.Blue, -150), sat+0.08)
	syntaxCyan := rainbeau.AdjustSaturation(rainbeau.ShiftHue(c.Blue, -30), sat+0.03)
	syntaxSapphire := rainbeau.AdjustSaturation(rainbeau.ShiftHue(c.Blue, 40), sat+0.02)
	syntaxProperty := rainbeau.AdjustSaturation(rainbeau.MixColors(c.Accent1, c.Blue, 0.35), sat)
	syntaxFlamingo := rainbeau.AdjustSaturation(rainbeau.MixColors(c.Red, c.Accent1, 0.40), sat+0.03)
	syntaxPink := rainbeau.AdjustSaturation(rainbeau.ShiftHue(c.Blue, 120), sat+0.04)
	syntaxRosewater := rainbeau.MixColors(c.Text, c.Accent1, 0.15)
	syntaxParam := rainbeau.MixColors(c.Red, c.Text, 0.40)

	colors := []string{syntaxKeyword, syntaxType, syntaxMethod, syntaxCyan, syntaxSapphire, syntaxProperty, syntaxFlamingo, syntaxPink, syntaxRosewater, syntaxParam}
	for i, color := range colors {
		colors[i] = rainbeau.EnsureContrast(color, c.Bg0, minContrast)
	}
	colors = rainbeau.EnsureDistinct(colors, c.Bg0)
	for i, color := range colors {
		colors[i] = rainbeau.EnsureContrast(color, c.Bg0, minContrast)
	}

	return replaceTokens(`-- Auto-generated by Rainbeau — do not edit manually
-- Theme: @@theme@@
return {
  {
    "catppuccin/nvim",
    name = "catppuccin",
    lazy = false,
    priority = 1000,
    config = function(_, opts)
      require("catppuccin").setup(opts)
      vim.cmd.colorscheme("catppuccin")
    end,
    opts = {
      flavour = "mocha",
      integrations = { lsp = true, treesitter = true },
      color_overrides = {
        mocha = {
          base = "@@bg0@@",
          mantle = "@@bg1@@",
          crust = "@@bg0@@",
          surface0 = "@@bg2@@",
          surface1 = "@@bg3@@",
          surface2 = "@@inactive@@",
          overlay0 = "@@textDim@@",
          overlay1 = "@@overlay1@@",
          overlay2 = "@@border@@",
          text = "@@text@@",
          subtext0 = "@@textDim@@",
          subtext1 = "@@subtext1@@",

          lavender = "@@syntaxProperty@@",
          blue = "@@blue@@",
          sapphire = "@@syntaxSapphire@@",
          sky = "@@syntaxCyan@@",
          teal = "@@syntaxType@@",
          green = "@@green@@",
          yellow = "@@syntaxMethod@@",
          peach = "@@accent2@@",
          maroon = "@@syntaxParam@@",
          red = "@@red@@",
          mauve = "@@syntaxKeyword@@",
          pink = "@@syntaxPink@@",
          flamingo = "@@syntaxFlamingo@@",
          rosewater = "@@syntaxRosewater@@",
        },
      },
      custom_highlights = function(colors)
        return {
          ["@lsp.type.keyword.cs"] = { fg = colors.mauve },
          ["@lsp.type.class.cs"] = { fg = colors.teal },
          ["@lsp.type.struct.cs"] = { fg = colors.teal, bold = true },
          ["@lsp.type.interface.cs"] = { fg = colors.sky, bold = true, italic = true },
          ["@lsp.type.enum.cs"] = { fg = colors.sapphire },
          ["@lsp.type.enumMember.cs"] = { fg = colors.flamingo },
          ["@lsp.type.typeParameter.cs"] = { fg = colors.teal, italic = true },
          ["@lsp.type.namespace.cs"] = { fg = colors.subtext0, italic = true },
          ["@lsp.type.method.cs"] = { fg = colors.yellow },
          ["@lsp.type.extensionMethodName.cs"] = { fg = colors.yellow, bold = true },
          ["@lsp.type.property.cs"] = { fg = colors.lavender },
          ["@lsp.type.field.cs"] = { fg = colors.lavender, italic = true },
          ["@lsp.type.staticField.cs"] = { fg = colors.peach, bold = true },
          ["@lsp.type.parameter.cs"] = { fg = colors.maroon, italic = true },
          ["@lsp.type.variable.cs"] = { fg = colors.text },
          ["@lsp.type.local.cs"] = { fg = colors.text },
          ["@lsp.type.delegate.cs"] = { fg = colors.flamingo, italic = true },
          ["@lsp.type.event.cs"] = { fg = colors.flamingo, bold = true },
          ["@lsp.type.string.cs"] = { fg = colors.green },
          ["@lsp.type.number.cs"] = { fg = colors.peach },
          ["@lsp.type.operator.cs"] = { fg = colors.sky },

          ["@keyword"] = { fg = colors.mauve },
          ["@type"] = { fg = colors.teal },
          ["@type.builtin"] = { fg = colors.teal, bold = true },
          ["@function"] = { fg = colors.yellow },
          ["@function.method"] = { fg = colors.yellow },
          ["@function.builtin"] = { fg = colors.yellow, italic = true },
          ["@constructor"] = { fg = colors.sapphire },
          ["@string"] = { fg = colors.green },
          ["@number"] = { fg = colors.peach },
          ["@variable"] = { fg = colors.text },
          ["@variable.parameter"] = { fg = colors.maroon, italic = true },
          ["@property"] = { fg = colors.lavender },
          ["@operator"] = { fg = colors.sky },
          ["@punctuation"] = { fg = colors.rosewater },
          ["@comment"] = { fg = colors.overlay0, italic = true },
          ["@constant"] = { fg = colors.peach },
          ["@constant.builtin"] = { fg = colors.peach, bold = true },
          ["@tag"] = { fg = colors.flamingo },
          ["@tag.attribute"] = { fg = colors.yellow },
          ["@namespace"] = { fg = colors.subtext0, italic = true },
        }
      end,
    },
  },
  {
    "LazyVim/LazyVim",
    opts = { colorscheme = "catppuccin" },
  },
}
`, map[string]string{
		"theme":           theme.Name,
		"bg0":             c.Bg0,
		"bg1":             c.Bg1,
		"bg2":             c.Bg2,
		"bg3":             c.Bg3,
		"inactive":        c.Inactive,
		"textDim":         p.TextDim,
		"overlay1":        rainbeau.MixColors(p.TextDim, p.Text, 0.2),
		"border":          p.Border,
		"text":            p.Text,
		"subtext1":        rainbeau.MixColors(p.TextDim, p.Text, 0.4),
		"syntaxProperty":  colors[5],
		"blue":            p.Blue,
		"syntaxSapphire":  colors[4],
		"syntaxCyan":      colors[3],
		"syntaxType":      colors[1],
		"green":           p.Green,
		"syntaxMethod":    colors[2],
		"accent2":         p.Accent2,
		"syntaxParam":     colors[9],
		"red":             p.Red,
		"syntaxKeyword":   colors[0],
		"syntaxPink":      colors[7],
		"syntaxFlamingo":  colors[6],
		"syntaxRosewater": colors[8],
	})
}
