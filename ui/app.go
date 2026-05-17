package ui

type App struct {
	build  *BuildOwner
	rootRO RenderObject
	size   Size
}

func NewApp(root Widget, opts ...Option) *App {
	owner := NewBuildOwner()
	owner.Mount(Provider[Theme]{Value: DefaultTheme(), ChildWidget: root})
	return &App{build: owner}
}

func (a *App) UpdateRoot(root Widget) {
	a.build.UpdateRoot(Provider[Theme]{Value: DefaultTheme(), ChildWidget: root})
}
func (a *App) Send(Event) {}

func (a *App) Pump(size Size) {
	a.size = size
	a.build.BuildScope()
	a.rootRO = findRenderObject(a.build.Root())
	if a.rootRO != nil {
		a.rootRO.Layout(LayoutContext{}, Tight(size))
	}
}

func (a *App) Paint(p *Painter) {
	if a.rootRO != nil {
		a.rootRO.Paint(p, Offset{})
	}
}

type Option func(*options)
type options struct{}
