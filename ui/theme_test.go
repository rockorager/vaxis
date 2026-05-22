package ui

import (
	"context"
	"testing"
)

type fakeColorQuerier struct {
	foreground Color
	background Color
}

func (q *fakeColorQuerier) QueryForeground(context.Context) Color {
	return q.foreground
}

func (q *fakeColorQuerier) QueryBackground(context.Context) Color {
	return q.background
}

func TestThemeFromTerminalDerivesRGBTheme(t *testing.T) {
	q := &fakeColorQuerier{
		foreground: RGB(200, 210, 220),
		background: RGB(10, 20, 30),
	}
	theme := themeFromTerminal(context.Background(), q)
	if got := theme.Foreground; got != q.foreground {
		t.Fatalf("text foreground = %#v, want terminal foreground %#v", got, q.foreground)
	}
	if got := theme.Background; got != q.background {
		t.Fatalf("background = %#v, want terminal background %#v", got, q.background)
	}
	if got := theme.Mode; got != DarkTheme {
		t.Fatalf("mode = %#v, want dark", got)
	}
	if theme.Surface == 0 || theme.Surface == theme.Background {
		t.Fatalf("surface = %#v, want derived non-background color", theme.Surface)
	}
	if theme.Primary == 0 {
		t.Fatalf("primary = %#v, want derived color", theme.Primary)
	}
	if got := contrastRatio(theme.Primary, theme.Foreground); got < 3 {
		t.Fatalf("primary contrast = %.2f, want readable with foreground", got)
	}
}

func TestDefaultThemeExposesDefaultBlueInPalette(t *testing.T) {
	if got, want := DefaultTheme().Palette.Blue.Tone500, DefaultBaseColors().Blue; got != want {
		t.Fatalf("palette blue tone 500 = %#v, want default blue %#v", got, want)
	}
}

func TestDefaultPaletteTone500SitsBetweenNeighbors(t *testing.T) {
	palette := DefaultPalette()
	scales := map[string]ColorScale{
		"red":     palette.Red,
		"green":   palette.Green,
		"yellow":  palette.Yellow,
		"blue":    palette.Blue,
		"magenta": palette.Magenta,
		"cyan":    palette.Cyan,
	}
	for name, scale := range scales {
		tone400 := colorLuminance(scale.Tone400)
		tone500 := colorLuminance(scale.Tone500)
		tone600 := colorLuminance(scale.Tone600)
		if tone400 <= tone500 || tone500 <= tone600 {
			t.Fatalf("%s luminance = 400:%.3f 500:%.3f 600:%.3f, want 500 between neighbors", name, tone400, tone500, tone600)
		}
	}
}

func TestDefaultThemePrimaryUsesDesignedBlueFillTone(t *testing.T) {
	theme := DefaultTheme()
	if got, want := theme.Primary, theme.Palette.Blue.Tone700; got != want {
		t.Fatalf("primary = %#v, want blue tone 700 %#v", got, want)
	}
	if got, want := theme.PrimaryHovered, theme.Palette.Blue.Tone600; got != want {
		t.Fatalf("primary hovered = %#v, want blue tone 600 %#v", got, want)
	}
	if got, want := theme.PrimaryPressed, theme.Palette.Blue.Tone800; got != want {
		t.Fatalf("primary pressed = %#v, want blue tone 800 %#v", got, want)
	}
	if got := contrastRatio(theme.Primary, theme.Foreground); got < 3 {
		t.Fatalf("primary foreground contrast = %.2f, want readable fill", got)
	}
	if got := contrastRatio(theme.Background, theme.PrimaryText); got < 3 {
		t.Fatalf("primary text contrast = %.2f, want readable text", got)
	}
	if theme.Selection == theme.PrimaryHovered || theme.Selection == theme.Primary {
		t.Fatalf("primary, primary hovered, and selection should be distinct: primary=%#v hovered=%#v selection=%#v", theme.Primary, theme.PrimaryHovered, theme.Selection)
	}
	if got, want := theme.Selection, theme.Palette.Blue.Tone800; got != want {
		t.Fatalf("selection = %#v, want blue tone 800 %#v", got, want)
	}
}

func TestThemeSurfaceStatesMoveInInteractionOrder(t *testing.T) {
	dark := DefaultTheme()
	if !(colorLuminance(dark.SurfacePressed) < colorLuminance(dark.Surface) &&
		colorLuminance(dark.Surface) < colorLuminance(dark.SurfaceHovered)) {
		t.Fatalf("dark surface luminance order = pressed %.3f surface %.3f hovered %.3f, want pressed < surface < hovered",
			colorLuminance(dark.SurfacePressed),
			colorLuminance(dark.Surface),
			colorLuminance(dark.SurfaceHovered))
	}

	light := ThemeFromPalette(DefaultPalette(), LightTheme)
	if !(colorLuminance(light.SurfaceHovered) < colorLuminance(light.Surface) &&
		colorLuminance(light.Surface) < colorLuminance(light.SurfacePressed)) {
		t.Fatalf("light surface luminance order = hovered %.3f surface %.3f pressed %.3f, want hovered < surface < pressed",
			colorLuminance(light.SurfaceHovered),
			colorLuminance(light.Surface),
			colorLuminance(light.SurfacePressed))
	}
}

func TestDefaultThemeAccentUsesMagentaScale(t *testing.T) {
	theme := DefaultTheme()
	cyanAccent := readableAccent(theme.Background, theme.Foreground, theme.Palette.Cyan, theme.Mode)
	if got, want := theme.Accent, theme.Palette.Magenta.Tone700; got != want {
		t.Fatalf("accent = %#v, want magenta tone 700 %#v", got, want)
	}
	if theme.Accent == cyanAccent {
		t.Fatalf("accent = %#v, want distinct from cyan-derived accent", theme.Accent)
	}
	if got := contrastRatio(theme.Accent, theme.Foreground); got < 3 {
		t.Fatalf("accent foreground contrast = %.2f, want readable fill", got)
	}
	if got := contrastRatio(theme.Background, theme.AccentText); got < 3 {
		t.Fatalf("accent text contrast = %.2f, want readable text", got)
	}
}

func TestDefaultThemeStatusColorsUseSemanticScales(t *testing.T) {
	theme := DefaultTheme()
	if got, want := theme.Success, theme.Palette.Green.Tone700; got != want {
		t.Fatalf("success = %#v, want green tone 700 %#v", got, want)
	}
	if got, want := theme.Warning, theme.Palette.Yellow.Tone700; got != want {
		t.Fatalf("warning = %#v, want yellow tone 700 %#v", got, want)
	}
	if got, want := theme.Danger, theme.Palette.Red.Tone700; got != want {
		t.Fatalf("danger = %#v, want red tone 700 %#v", got, want)
	}
	if got := contrastRatio(theme.Success, theme.Foreground); got < 3 {
		t.Fatalf("success foreground contrast = %.2f, want readable fill", got)
	}
	if got := contrastRatio(theme.Warning, theme.Foreground); got < 3 {
		t.Fatalf("warning foreground contrast = %.2f, want readable fill", got)
	}
	if got := contrastRatio(theme.Danger, theme.Foreground); got < 3 {
		t.Fatalf("danger foreground contrast = %.2f, want readable fill", got)
	}
	if got := contrastRatio(theme.Background, theme.SuccessText); got < 3 {
		t.Fatalf("success text contrast = %.2f, want readable text", got)
	}
	if got := contrastRatio(theme.Background, theme.WarningText); got < 3 {
		t.Fatalf("warning text contrast = %.2f, want readable text", got)
	}
	if got := contrastRatio(theme.Background, theme.DangerText); got < 3 {
		t.Fatalf("danger text contrast = %.2f, want readable text", got)
	}
}

func TestPrimaryFallsBackWhenPreferredToneMatchesBackground(t *testing.T) {
	palette := DefaultPalette()
	palette.Blue.Tone700 = palette.Neutral.Tone950
	theme := ThemeFromPalette(palette, DarkTheme)
	if got, invisible := theme.Primary, palette.Blue.Tone700; got == invisible {
		t.Fatalf("primary = %#v, want fallback away from invisible preferred tone", got)
	}
}

func TestThemeFromTerminalFallsBackForMissingColors(t *testing.T) {
	fallback := DefaultTheme()
	theme := themeFromTerminal(context.Background(), &fakeColorQuerier{})
	if theme != fallback {
		t.Fatalf("theme = %#v, want fallback %#v", theme, fallback)
	}
}

func TestThemeSetSwitchesOnColorThemeUpdate(t *testing.T) {
	themeSet := ThemeSet{
		Light: Theme{Foreground: RGB(1, 2, 3)},
		Dark:  Theme{Foreground: RGB(4, 5, 6)},
	}
	app := NewApp(Text{Value: "x"}, WithThemeSet(themeSet))
	app.Pump(Size{Width: 1, Height: 1})
	p := NewPainter(Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Foreground; got != themeSet.Dark.Foreground {
		t.Fatalf("initial foreground = %#v, want dark foreground", got)
	}

	app.Send(ColorThemeUpdate{Mode: LightMode})
	app.Pump(Size{Width: 1, Height: 1})
	p = NewPainter(Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Foreground; got != themeSet.Light.Foreground {
		t.Fatalf("updated foreground = %#v, want light foreground", got)
	}
}

func TestStaticThemeIgnoresColorThemeUpdate(t *testing.T) {
	theme := Theme{Foreground: RGB(4, 5, 6)}
	app := NewApp(Text{Value: "x"}, WithTheme(theme))
	app.Pump(Size{Width: 1, Height: 1})

	app.Send(ColorThemeUpdate{Mode: LightMode})
	app.Pump(Size{Width: 1, Height: 1})
	p := NewPainter(Size{Width: 1, Height: 1})
	app.Paint(p)
	if got := p.Cell(0, 0).Foreground; got != theme.Foreground {
		t.Fatalf("foreground = %#v, want static theme foreground", got)
	}
}
