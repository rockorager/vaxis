package ui

import (
	"context"
	"math"
)

// ThemeMode selects how a palette is mapped to semantic UI colors.
type ThemeMode int

const (
	// DarkTheme maps dark neutral tones to backgrounds and light tones to text.
	DarkTheme ThemeMode = iota
	// LightTheme maps light neutral tones to backgrounds and dark tones to text.
	LightTheme
)

// BaseColors is the compact color input used to generate a Palette.
type BaseColors struct {
	Black   Color
	Red     Color
	Green   Color
	Yellow  Color
	Blue    Color
	Magenta Color
	Cyan    Color
	White   Color
}

// ColorScale contains generated tones for one color family. Lower tones are
// lighter, and higher tones are darker.
type ColorScale struct {
	Tone50  Color
	Tone100 Color
	Tone200 Color
	Tone300 Color
	Tone400 Color
	Tone500 Color
	Tone600 Color
	Tone700 Color
	Tone800 Color
	Tone900 Color
	Tone950 Color
}

// Palette is the color scale available to a Theme.
type Palette struct {
	Neutral ColorScale
	Red     ColorScale
	Green   ColorScale
	Yellow  ColorScale
	Blue    ColorScale
	Magenta ColorScale
	Cyan    ColorScale
}

// Theme contains semantic colors used by built-in and custom widgets.
type Theme struct {
	// Palette is the source color scale used to generate the semantic colors.
	Palette Palette
	// Mode is the light/dark mapping used to generate the semantic colors.
	Mode ThemeMode

	// Background is the app's base fill color.
	Background Color
	// Foreground is the default readable text/icon color for Background,
	// Surface*, Primary*, Accent, and status fills unless a component documents a
	// stronger pairing.
	Foreground Color

	// Surface is the default panel/control fill.
	Surface Color
	// SurfaceRaised is an elevated panel/popover fill, such as dialogs and
	// command palettes.
	SurfaceRaised Color
	// SurfaceHovered is the interactive hover/focus fill for surface controls.
	SurfaceHovered Color
	// SurfacePressed is the active/pressed fill for surface controls.
	SurfacePressed Color

	// Primary is the primary emphasis fill. Pair it with Foreground for text on
	// the fill; do not pair it with PrimaryText.
	Primary Color
	// PrimaryText is a primary-colored text/icon foreground for use on normal
	// backgrounds and surfaces. It is not intended as text on Primary fills.
	PrimaryText Color
	// PrimaryHovered is the hover/focus fill for primary controls. Pair it with
	// Foreground for text on the fill.
	PrimaryHovered Color
	// PrimaryPressed is the active/pressed fill for primary controls. Pair it with
	// Foreground for text on the fill.
	PrimaryPressed Color

	// Accent is a secondary emphasis fill. Pair it with Foreground for text on the
	// fill; do not pair it with AccentText.
	Accent Color
	// AccentText is an accent-colored text/icon foreground for use on normal
	// backgrounds and surfaces. It is not intended as text on Accent fills.
	AccentText Color

	// Success is a success-state fill. Pair it with Foreground for text on the
	// fill; do not pair it with SuccessText.
	Success Color
	// SuccessText is a success-colored text/icon foreground for use on normal
	// backgrounds and surfaces. It is not intended as text on Success fills.
	SuccessText Color
	// Warning is a warning-state fill. Pair it with Foreground for text on the
	// fill; do not pair it with WarningText.
	Warning Color
	// WarningText is a warning-colored text/icon foreground for use on normal
	// backgrounds and surfaces. It is not intended as text on Warning fills.
	WarningText Color
	// Danger is a danger/error-state fill. Pair it with Foreground for text on the
	// fill; do not pair it with DangerText.
	Danger Color
	// DangerText is a danger-colored text/icon foreground for use on normal
	// backgrounds and surfaces. It is not intended as text on Danger fills.
	DangerText Color

	// MutedForeground is low-emphasis readable text for secondary information.
	MutedForeground Color
	// DisabledForeground is low-contrast text for unavailable controls.
	DisabledForeground Color

	// Selection is the fill used behind selected text. Pair it with Foreground.
	Selection Color

	// Border is a subtle divider/border color.
	Border Color
}

// ThemeSet contains resolved themes for light and dark appearances.
type ThemeSet struct {
	Light Theme
	Dark  Theme
}

// Theme returns the theme matching mode.
func (s ThemeSet) Theme(mode ThemeMode) Theme {
	if mode == LightTheme {
		if s.Light != (Theme{}) {
			return s.Light
		}
		return s.Dark
	}
	if s.Dark != (Theme{}) {
		return s.Dark
	}
	return s.Light
}

// DefaultBaseColors returns the built-in vaxis/ui base colors.
func DefaultBaseColors() BaseColors {
	return BaseColors{
		Black:   RGB(12, 14, 18),
		Red:     RGB(255, 118, 118),
		Green:   RGB(83, 196, 149),
		Yellow:  RGB(229, 184, 91),
		Blue:    RGB(80, 150, 255),
		Magenta: RGB(181, 137, 230),
		Cyan:    RGB(78, 196, 184),
		White:   RGB(245, 246, 250),
	}
}

// DefaultPalette returns the built-in vaxis/ui color palette.
func DefaultPalette() Palette {
	return PaletteFromBaseColors(DefaultBaseColors())
}

// DefaultTheme returns the built-in vaxis/ui theme.
func DefaultTheme() Theme {
	return ThemeFromPalette(DefaultPalette(), DarkTheme)
}

// DefaultThemeSet returns the built-in vaxis/ui light and dark themes.
func DefaultThemeSet() ThemeSet {
	return ThemeSetFromPalette(DefaultPalette())
}

// ThemeSetFromBaseColors generates light and dark themes from base colors.
func ThemeSetFromBaseColors(base BaseColors) ThemeSet {
	return ThemeSetFromPalette(PaletteFromBaseColors(base))
}

// PaletteFromBaseColors generates a color scale from base colors.
func PaletteFromBaseColors(base BaseColors) Palette {
	base = fillBaseColors(base)
	return Palette{
		Neutral: neutralScale(base.Black, base.White),
		Red:     colorScale(base.Red, base.Black, base.White),
		Green:   colorScale(base.Green, base.Black, base.White),
		Yellow:  colorScale(base.Yellow, base.Black, base.White),
		Blue:    colorScale(base.Blue, base.Black, base.White),
		Magenta: colorScale(base.Magenta, base.Black, base.White),
		Cyan:    colorScale(base.Cyan, base.Black, base.White),
	}
}

// ThemeFromPalette maps a palette to contrast-aware semantic colors.
func ThemeFromPalette(p Palette, mode ThemeMode) Theme {
	return themeFromPalette(p, mode, 0, 0)
}

// ThemeSetFromPalette maps one palette to light and dark semantic themes.
func ThemeSetFromPalette(p Palette) ThemeSet {
	return ThemeSet{
		Light: ThemeFromPalette(p, LightTheme),
		Dark:  ThemeFromPalette(p, DarkTheme),
	}
}

func themeModeFromColorThemeMode(mode ColorThemeMode) (ThemeMode, bool) {
	switch mode {
	case LightMode:
		return LightTheme, true
	case DarkMode:
		return DarkTheme, true
	default:
		return DarkTheme, false
	}
}

type terminalColorQuerier interface {
	QueryForeground(context.Context) Color
	QueryBackground(context.Context) Color
}

type terminalPaletteQuerier interface {
	terminalColorQuerier
	QueryColor(context.Context, uint8) Color
}

func themeFromTerminal(ctx context.Context, q terminalColorQuerier) Theme {
	if q == nil {
		return DefaultTheme()
	}
	fg := q.QueryForeground(ctx)
	bg := q.QueryBackground(ctx)
	base := baseColorsFromTerminal(ctx, q, fg, bg)
	mode := DarkTheme
	if colorLuminance(bg) > colorLuminance(fg) {
		mode = LightTheme
	}
	return themeFromPalette(PaletteFromBaseColors(base), mode, bg, fg)
}

func baseColorsFromTerminal(ctx context.Context, q terminalColorQuerier, fg, bg Color) BaseColors {
	base := DefaultBaseColors()
	if pq, ok := q.(terminalPaletteQuerier); ok {
		if c := pq.QueryColor(ctx, 1); c != 0 {
			base.Red = c
		}
		if c := pq.QueryColor(ctx, 2); c != 0 {
			base.Green = c
		}
		if c := pq.QueryColor(ctx, 3); c != 0 {
			base.Yellow = c
		}
		if c := pq.QueryColor(ctx, 4); c != 0 {
			base.Blue = c
		}
		if c := pq.QueryColor(ctx, 5); c != 0 {
			base.Magenta = c
		}
		if c := pq.QueryColor(ctx, 6); c != 0 {
			base.Cyan = c
		}
		if c := pq.QueryColor(ctx, 0); c != 0 {
			base.Black = c
		}
		if c := pq.QueryColor(ctx, 7); c != 0 {
			base.White = c
		}
	}
	if fg != 0 && bg != 0 {
		if colorLuminance(bg) > colorLuminance(fg) {
			base.Black = fg
			base.White = bg
		} else {
			base.Black = bg
			base.White = fg
		}
	}
	return base
}

func themeFromPalette(p Palette, mode ThemeMode, background, foreground Color) Theme {
	if background == 0 {
		if mode == LightTheme {
			background = p.Neutral.Tone50
		} else {
			background = p.Neutral.Tone950
		}
	}
	if foreground == 0 {
		if mode == LightTheme {
			foreground = p.Neutral.Tone950
		} else {
			foreground = p.Neutral.Tone50
		}
	}

	surface, surfaceRaised, surfaceHovered, surfacePressed := paletteSurfaces(p, mode)
	primary := readablePrimary(background, foreground, p.Blue, mode)
	primaryHovered := readablePrimaryHovered(background, foreground, p.Blue, mode)
	primaryPressed := readablePrimaryPressed(background, foreground, p.Blue, mode)
	accent := readableAccent(background, foreground, p.Magenta, mode)
	success := readableStatus(background, foreground, p.Green, mode)
	warning := readableStatus(background, foreground, p.Yellow, mode)
	danger := readableStatus(background, foreground, p.Red, mode)
	selection := readableSelection(background, foreground, p.Blue, mode)
	border := readableSoftColor(background, foreground, p.Neutral, mode, 2.0)

	return Theme{
		Palette: p,
		Mode:    mode,

		Background: background,
		Foreground: foreground,

		Surface:        surface,
		SurfaceRaised:  surfaceRaised,
		SurfaceHovered: surfaceHovered,
		SurfacePressed: surfacePressed,

		Primary:        primary,
		PrimaryText:    readableTextColor(background, p.Blue, mode),
		PrimaryHovered: primaryHovered,
		PrimaryPressed: primaryPressed,

		Accent:     accent,
		AccentText: readableTextColor(background, p.Magenta, mode),

		Success:     success,
		SuccessText: readableTextColor(background, p.Green, mode),
		Warning:     warning,
		WarningText: readableTextColor(background, p.Yellow, mode),
		Danger:      danger,
		DangerText:  readableTextColor(background, p.Red, mode),

		MutedForeground:    readableSoftColor(background, foreground, p.Neutral, mode, 3.0),
		DisabledForeground: readableSoftColor(background, foreground, p.Neutral, mode, 2.0),

		Selection: selection,

		Border: border,
	}
}

func fillBaseColors(base BaseColors) BaseColors {
	def := DefaultBaseColors()
	if base.Black == 0 {
		base.Black = def.Black
	}
	if base.Red == 0 {
		base.Red = def.Red
	}
	if base.Green == 0 {
		base.Green = def.Green
	}
	if base.Yellow == 0 {
		base.Yellow = def.Yellow
	}
	if base.Blue == 0 {
		base.Blue = def.Blue
	}
	if base.Magenta == 0 {
		base.Magenta = def.Magenta
	}
	if base.Cyan == 0 {
		base.Cyan = def.Cyan
	}
	if base.White == 0 {
		base.White = def.White
	}
	return base
}

func neutralScale(black, white Color) ColorScale {
	return ColorScale{
		Tone50:  white,
		Tone100: blendOr(white, black, 5, white),
		Tone200: blendOr(white, black, 12, white),
		Tone300: blendOr(white, black, 22, white),
		Tone400: blendOr(white, black, 35, white),
		Tone500: blendOr(white, black, 50, white),
		Tone600: blendOr(white, black, 65, black),
		Tone700: blendOr(white, black, 78, black),
		Tone800: blendOr(white, black, 88, black),
		Tone900: blendOr(white, black, 95, black),
		Tone950: black,
	}
}

func colorScale(base, black, white Color) ColorScale {
	if scale, ok := oklchColorScale(base, black, white); ok {
		return scale
	}
	return ColorScale{
		Tone50:  blendOr(white, base, 10, base),
		Tone100: blendOr(white, base, 20, base),
		Tone200: blendOr(white, base, 35, base),
		Tone300: blendOr(white, base, 55, base),
		Tone400: blendOr(white, base, 75, base),
		Tone500: base,
		Tone600: blendOr(base, black, 15, base),
		Tone700: blendOr(base, black, 30, base),
		Tone800: blendOr(base, black, 45, base),
		Tone900: blendOr(base, black, 60, base),
		Tone950: blendOr(base, black, 75, base),
	}
}

func paletteSurfaces(p Palette, mode ThemeMode) (surface, raised, hovered, pressed Color) {
	if mode == LightTheme {
		return p.Neutral.Tone100, p.Neutral.Tone200, p.Neutral.Tone200, p.Neutral.Tone50
	}
	return p.Neutral.Tone900, p.Neutral.Tone800, p.Neutral.Tone800, p.Neutral.Tone950
}

func readableAccent(bg, fg Color, scale ColorScale, mode ThemeMode) Color {
	if mode == LightTheme {
		return readableFill(bg, fg, 3.0, scale.Tone300, scale.Tone200, scale.Tone400, scale.Tone500, scale.Tone600)
	}
	return readableFill(bg, fg, 3.0, scale.Tone700, scale.Tone800, scale.Tone600, scale.Tone500, scale.Tone900)
}

func readableStatus(bg, fg Color, scale ColorScale, mode ThemeMode) Color {
	if mode == LightTheme {
		return readableFill(bg, fg, 3.0, scale.Tone300, scale.Tone200, scale.Tone400, scale.Tone500, scale.Tone600)
	}
	return readableFill(bg, fg, 3.0, scale.Tone700, scale.Tone800, scale.Tone600, scale.Tone500, scale.Tone900)
}

func readableTextColor(bg Color, scale ColorScale, mode ThemeMode) Color {
	if mode == LightTheme {
		return preferredColor(bg, 3.0, scale.Tone700, scale.Tone800, scale.Tone600, scale.Tone900, scale.Tone500)
	}
	return preferredColor(bg, 3.0, scale.Tone400, scale.Tone300, scale.Tone500, scale.Tone200, scale.Tone600)
}

func readablePrimary(bg, fg Color, scale ColorScale, mode ThemeMode) Color {
	if mode == LightTheme {
		return readableFill(bg, fg, 3.0, scale.Tone300, scale.Tone200, scale.Tone400, scale.Tone500, scale.Tone100)
	}
	return readableFill(bg, fg, 3.0, scale.Tone700, scale.Tone800, scale.Tone600, scale.Tone500, scale.Tone900)
}

func readablePrimaryHovered(bg, fg Color, scale ColorScale, mode ThemeMode) Color {
	if mode == LightTheme {
		return readableFill(bg, fg, 3.0, scale.Tone200, scale.Tone300, scale.Tone100, scale.Tone400, scale.Tone500)
	}
	return readableFill(bg, fg, 3.0, scale.Tone500, scale.Tone600, scale.Tone400, scale.Tone700, scale.Tone800)
}

func readablePrimaryPressed(bg, fg Color, scale ColorScale, mode ThemeMode) Color {
	if mode == LightTheme {
		return readableFill(bg, fg, 3.0, scale.Tone400, scale.Tone300, scale.Tone500, scale.Tone200, scale.Tone600)
	}
	return readableFill(bg, fg, 3.0, scale.Tone800, scale.Tone900, scale.Tone700, scale.Tone600, scale.Tone500)
}

func readableSelection(bg, fg Color, scale ColorScale, mode ThemeMode) Color {
	if mode == LightTheme {
		return readableFill(bg, fg, 3.0, scale.Tone100, scale.Tone200, scale.Tone300, scale.Tone400, scale.Tone500)
	}
	return readableFill(bg, fg, 3.0, scale.Tone800, scale.Tone900, scale.Tone950, scale.Tone700, scale.Tone600)
}

func readableFill(bg, fg Color, minForegroundContrast float64, candidates ...Color) Color {
	for _, c := range candidates {
		if c != 0 && contrastRatio(fg, c) >= minForegroundContrast && contrastRatio(bg, c) >= 1.2 {
			return c
		}
	}
	best := readableColor(fg, candidates...)
	if contrastRatio(bg, best) < 1.2 {
		return readableColor(bg, candidates...)
	}
	return best
}

func readableSoftColor(bg, fg Color, neutral ColorScale, mode ThemeMode, minContrast float64) Color {
	candidates := []Color{neutral.Tone600, neutral.Tone500, neutral.Tone700, neutral.Tone400, neutral.Tone300, fg}
	if mode == LightTheme {
		candidates = []Color{neutral.Tone400, neutral.Tone500, neutral.Tone300, neutral.Tone600, neutral.Tone700, fg}
	}
	for _, c := range candidates {
		if c != 0 && c != fg && contrastRatio(bg, c) >= minContrast {
			return c
		}
	}
	return readableColor(bg, fg, neutral.Tone50, neutral.Tone950)
}

func readableColor(bg Color, candidates ...Color) Color {
	best := Color(0)
	bestRatio := -1.0
	for _, c := range candidates {
		if c == 0 {
			continue
		}
		ratio := contrastRatio(bg, c)
		if ratio > bestRatio {
			best = c
			bestRatio = ratio
		}
	}
	return best
}

func preferredColor(bg Color, minContrast float64, candidates ...Color) Color {
	for _, c := range candidates {
		if c != 0 && contrastRatio(bg, c) >= minContrast {
			return c
		}
	}
	return readableColor(bg, candidates...)
}

func blendOr(a, b Color, percentB int, fallback Color) Color {
	c, ok := blendColor(a, b, percentB)
	if !ok {
		return fallback
	}
	return c
}

func blendColor(a, b Color, percentB int) (Color, bool) {
	ap := a.Params()
	bp := b.Params()
	if len(ap) != 3 || len(bp) != 3 {
		return 0, false
	}
	aa := rgbToOKLab(float64(ap[0])/255, float64(ap[1])/255, float64(ap[2])/255)
	bb := rgbToOKLab(float64(bp[0])/255, float64(bp[1])/255, float64(bp[2])/255)
	t := float64(percentB) / 100
	return okLabToColor(okLab{
		l: aa.l + (bb.l-aa.l)*t,
		a: aa.a + (bb.a-aa.a)*t,
		b: aa.b + (bb.b-aa.b)*t,
	}), true
}

func contrastRatio(a, b Color) float64 {
	la := colorLuminance(a)
	lb := colorLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func colorLuminance(c Color) float64 {
	p := c.Params()
	if len(p) != 3 {
		return 0
	}
	r := linearColor(float64(p[0]) / 255)
	g := linearColor(float64(p[1]) / 255)
	b := linearColor(float64(p[2]) / 255)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func linearColor(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

type okLab struct {
	l float64
	a float64
	b float64
}

type okLCH struct {
	l float64
	c float64
	h float64
}

func oklchColorScale(base, black, white Color) (ColorScale, bool) {
	bp := base.Params()
	blackp := black.Params()
	whitep := white.Params()
	if len(bp) != 3 || len(blackp) != 3 || len(whitep) != 3 {
		return ColorScale{}, false
	}
	b := rgbToOKLCH(float64(bp[0])/255, float64(bp[1])/255, float64(bp[2])/255)
	blackLab := rgbToOKLab(float64(blackp[0])/255, float64(blackp[1])/255, float64(blackp[2])/255)
	whiteLab := rgbToOKLab(float64(whitep[0])/255, float64(whitep[1])/255, float64(whitep[2])/255)
	lo := min(blackLab.l, whiteLab.l)
	hi := max(blackLab.l, whiteLab.l)
	light := func(t, chroma float64) Color {
		l := b.l + (hi-b.l)*t
		return okLCHToColor(okLCH{l: l, c: b.c * chroma, h: b.h})
	}
	dark := func(t, chroma float64) Color {
		l := b.l + (lo-b.l)*t
		return okLCHToColor(okLCH{l: l, c: b.c * chroma, h: b.h})
	}
	return ColorScale{
		Tone50:  light(0.90, 0.25),
		Tone100: light(0.78, 0.35),
		Tone200: light(0.60, 0.55),
		Tone300: light(0.42, 0.75),
		Tone400: light(0.22, 0.90),
		Tone500: base,
		Tone600: dark(0.18, 0.95),
		Tone700: dark(0.36, 0.85),
		Tone800: dark(0.54, 0.70),
		Tone900: dark(0.72, 0.55),
		Tone950: dark(0.86, 0.40),
	}, true
}

func rgbToOKLCH(r, g, b float64) okLCH {
	lab := rgbToOKLab(r, g, b)
	return okLCH{
		l: lab.l,
		c: math.Hypot(lab.a, lab.b),
		h: math.Atan2(lab.b, lab.a),
	}
}

func rgbToOKLab(r, g, b float64) okLab {
	r = linearColor(r)
	g = linearColor(g)
	b = linearColor(b)

	l := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b

	l_ := math.Cbrt(l)
	m_ := math.Cbrt(m)
	s_ := math.Cbrt(s)

	return okLab{
		l: 0.2104542553*l_ + 0.7936177850*m_ - 0.0040720468*s_,
		a: 1.9779984951*l_ - 2.4285922050*m_ + 0.4505937099*s_,
		b: 0.0259040371*l_ + 0.7827717662*m_ - 0.8086757660*s_,
	}
}

func okLCHToColor(c okLCH) Color {
	return okLabToColor(okLab{
		l: c.l,
		a: math.Cos(c.h) * c.c,
		b: math.Sin(c.h) * c.c,
	})
}

func okLabToColor(c okLab) Color {
	l_ := c.l + 0.3963377774*c.a + 0.2158037573*c.b
	m_ := c.l - 0.1055613458*c.a - 0.0638541728*c.b
	s_ := c.l - 0.0894841775*c.a - 1.2914855480*c.b

	l := l_ * l_ * l_
	m := m_ * m_ * m_
	s := s_ * s_ * s_

	r := 4.0767416621*l - 3.3077115913*m + 0.2309699292*s
	g := -1.2684380046*l + 2.6097574011*m - 0.3413193965*s
	b := -0.0041960863*l - 0.7034186147*m + 1.7076147010*s

	return RGB(
		uint8(math.Round(clampThemeFloat(gammaEncode(r), 0, 1)*255)),
		uint8(math.Round(clampThemeFloat(gammaEncode(g), 0, 1)*255)),
		uint8(math.Round(clampThemeFloat(gammaEncode(b), 0, 1)*255)),
	)
}

func gammaEncode(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

func clampThemeFloat(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

const (
	defaultButtonMinWidth     = 5
	defaultListTileGap        = 1
	defaultListTileMinHeight  = 1
	defaultTextFieldMinWidth  = 10
	defaultButtonFocusLeft    = "["
	defaultButtonFocusRight   = "]"
	defaultButtonMouseShape   = MouseShapeClickable
	defaultListTileMouseShape = MouseShapeClickable
	defaultSegmentMouseShape  = MouseShapeClickable
)

// ButtonTheme contains derived styling and sizing defaults for Button.
type ButtonTheme struct {
	Normal         Style
	Focused        Style
	Hovered        Style
	FocusedHovered Style
	Pressed        Style
	Padding        Insets
	MinWidth       int
	Mouse          MouseShape
	FocusLeft      Character
	FocusRight     Character
}

// ProgressBarTheme contains derived styling defaults for ProgressBar.
type ProgressBarTheme struct {
	Filled Style
	Empty  Style
}

// SegmentedControlTheme contains derived styling defaults for SegmentedControl.
type SegmentedControlTheme struct {
	Normal          Style
	Focused         Style
	Hovered         Style
	Selected        Style
	SelectedHovered Style
	Disabled        Style
	Separator       Style
	Mouse           MouseShape
}

// ListTileTheme contains derived styling and sizing defaults for ListTile.
type ListTileTheme struct {
	Normal          Style
	Focused         Style
	Hovered         Style
	Selected        Style
	SelectedFocused Style
	SelectedHovered Style
	Disabled        Style
	Padding         Insets
	Gap             int
	MinHeight       int
	Mouse           MouseShape
}

// TextFieldTheme contains derived styling and sizing defaults for TextField and TextArea.
type TextFieldTheme struct {
	Normal      Style
	Focused     Style
	Placeholder Style
	Cursor      Style
	Selection   Style
	Padding     Insets
	MinWidth    int
}

// ScrollbarTheme contains derived styling defaults for Scrollbar.
type ScrollbarTheme struct {
	Thumb        Style
	Track        Style
	FocusedThumb Style
	FocusedTrack Style
}

func textStyle(theme Theme) Style {
	return Style{Foreground: theme.Foreground}
}

func buttonTheme(theme Theme) ButtonTheme {
	return ButtonTheme{
		Normal:         Style{Foreground: theme.Foreground, Background: theme.Surface},
		Focused:        Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered},
		Hovered:        Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered},
		FocusedHovered: Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered},
		Pressed:        Style{Foreground: theme.Foreground, Background: theme.PrimaryPressed},
		Padding:        Symmetric(1, 0),
		MinWidth:       defaultButtonMinWidth,
		Mouse:          defaultButtonMouseShape,
		FocusLeft:      Character{Grapheme: defaultButtonFocusLeft, Width: 1},
		FocusRight:     Character{Grapheme: defaultButtonFocusRight, Width: 1},
	}
}

func listTileTheme(theme Theme) ListTileTheme {
	return ListTileTheme{
		Normal:          Style{Foreground: theme.Foreground},
		Focused:         Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered},
		Hovered:         Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered},
		Selected:        Style{Foreground: theme.Foreground, Background: theme.Primary},
		SelectedFocused: Style{Foreground: theme.Foreground, Background: theme.Primary},
		SelectedHovered: Style{Foreground: theme.Foreground, Background: theme.PrimaryHovered},
		Disabled:        Style{Foreground: theme.DisabledForeground},
		Padding:         Symmetric(1, 0),
		Gap:             defaultListTileGap,
		MinHeight:       defaultListTileMinHeight,
		Mouse:           defaultListTileMouseShape,
	}
}

func segmentedControlTheme(theme Theme) SegmentedControlTheme {
	return SegmentedControlTheme{
		Normal:          Style{Foreground: theme.Foreground, Background: theme.Surface},
		Focused:         Style{UnderlineStyle: UnderlineSingle},
		Hovered:         Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered},
		Selected:        Style{Foreground: theme.Foreground, Background: theme.Primary},
		SelectedHovered: Style{Foreground: theme.Foreground, Background: theme.PrimaryHovered},
		Disabled:        Style{Foreground: theme.DisabledForeground},
		Separator:       Style{Foreground: theme.Border, Background: theme.Surface},
		Mouse:           defaultSegmentMouseShape,
	}
}

func progressBarTheme(theme Theme) ProgressBarTheme {
	return ProgressBarTheme{
		Filled: Style{Foreground: theme.Accent, Background: theme.Surface},
		Empty:  Style{Foreground: theme.Border, Background: theme.Surface},
	}
}

func textFieldTheme(theme Theme) TextFieldTheme {
	return TextFieldTheme{
		Normal:      Style{Foreground: theme.Foreground, Background: theme.Surface},
		Focused:     Style{Foreground: theme.Foreground, Background: theme.SurfaceHovered},
		Placeholder: Style{Foreground: theme.MutedForeground, Background: theme.Surface},
		Cursor:      Style{Foreground: theme.Background, Background: theme.Foreground},
		Selection:   Style{Foreground: theme.Foreground, Background: theme.Selection},
		Padding:     Symmetric(1, 0),
		MinWidth:    defaultTextFieldMinWidth,
	}
}

func scrollbarTheme(theme Theme) ScrollbarTheme {
	return ScrollbarTheme{
		Thumb:        Style{Background: theme.Border},
		Track:        Style{Background: theme.Surface},
		FocusedThumb: Style{Background: theme.AccentText},
		FocusedTrack: Style{Background: theme.SurfaceHovered},
	}
}
