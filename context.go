// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

import (
	"fmt"
	"image"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/internal/locale"
)

var envLocales []language.Tag

func init() {
	if locales := os.Getenv("GUIGUI_LOCALES"); locales != "" {
		for tag := range strings.SplitSeq(os.Getenv("GUIGUI_LOCALES"), ",") {
			l, err := language.Parse(strings.TrimSpace(tag))
			if err != nil {
				slog.Warn(fmt.Sprintf("invalid GUIGUI_LOCALES: %s", tag))
				continue
			}
			envLocales = append(envLocales, l)
		}
	}
}

var systemLocales []language.Tag

func init() {
	ls, err := locale.Locales()
	if err != nil {
		slog.Error(err.Error())
		return
	}
	systemLocales = ls
}

type Context struct {
	app     *app
	inBuild bool

	appScaleMinus1       float64
	defaultColorWarnOnce sync.Once
	locales              []language.Tag
	allLocales           []language.Tag
	frontLayer           int64
	envSource            EnvSource

	defaultProxyMethodCalled bool
	defaultTickMethodCalled  bool
}

func (c *Context) Scale() float64 {
	return c.DeviceScale() * c.AppScale()
}

func (c *Context) DeviceScale() float64 {
	return c.app.deviceScale
}

func (c *Context) AppScale() float64 {
	return c.appScaleMinus1 + 1
}

func (c *Context) SetAppScale(scale float64) {
	if c.appScaleMinus1 == scale-1 {
		return
	}
	c.appScaleMinus1 = scale - 1
	c.app.requestRedraw(c.app.bounds(), requestRedrawReasonAppScale, nil)
}

// ResolvedColorMode returns the color mode.
//
// ResolvedColorMode never returns [ebiten.ColorModeUnknown].
func (c *Context) ResolvedColorMode() ebiten.ColorMode {
	if mode := ebiten.WindowColorMode(); mode != ebiten.ColorModeUnknown {
		return mode
	}
	if mode := ebiten.SystemColorMode(); mode != ebiten.ColorModeUnknown {
		return mode
	}
	return ebiten.ColorModeLight
}

// ColorMode returns the color mode set by SetColorMode.
//
// ColorMode might return [ebiten.ColorModeUnknown] if the color mode is not set.
func (c *Context) ColorMode() ebiten.ColorMode {
	return ebiten.WindowColorMode()
}

// SetColorMode sets the color mode.
//
// If mode is [ebiten.ColorModeUnknown], SetColorMode specifies the default system color mode.
func (c *Context) SetColorMode(mode ebiten.ColorMode) {
	if mode == ebiten.WindowColorMode() {
		return
	}
	ebiten.SetWindowColorMode(mode)
	c.app.requestRebuild(c.app.root.widgetState(), requestRedrawReasonColorMode)
}

var (
	envColorModeStr = os.Getenv("GUIGUI_COLOR_MODE")
)

func init() {
	switch envColorModeStr {
	case "light":
		ebiten.SetWindowColorMode(ebiten.ColorModeLight)
	case "dark":
		ebiten.SetWindowColorMode(ebiten.ColorModeDark)
	case "":
	default:
		slog.Warn(fmt.Sprintf("invalid GUIGUI_COLOR_MODE: %s", envColorModeStr))
	}
}

func (c *Context) AppendLocales(locales []language.Tag) []language.Tag {
	if len(c.allLocales) == 0 {
		// App locales
		for _, l := range c.locales {
			if slices.Contains(c.allLocales, l) {
				continue
			}
			c.allLocales = append(c.allLocales, l)
		}
		// Env locales
		for _, l := range envLocales {
			if slices.Contains(c.allLocales, l) {
				continue
			}
			c.allLocales = append(c.allLocales, l)
		}
		// System locales
		for _, l := range systemLocales {
			if slices.Contains(c.allLocales, l) {
				continue
			}
			c.allLocales = append(c.allLocales, l)
		}
	}
	return append(locales, c.allLocales...)
}

func (c *Context) AppendAppLocales(locales []language.Tag) []language.Tag {
	origLen := len(locales)
	for _, l := range c.locales {
		if slices.Contains(locales[origLen:], l) {
			continue
		}
		locales = append(locales, l)
	}
	return locales
}

func (c *Context) SetAppLocales(locales []language.Tag) {
	if slices.Equal(c.locales, locales) {
		return
	}

	c.locales = slices.Delete(c.locales, 0, len(c.locales))
	c.locales = append(c.locales, locales...)
	c.allLocales = slices.Delete(c.allLocales, 0, len(c.allLocales))

	c.app.requestRedraw(c.app.bounds(), requestRedrawReasonLocale, nil)
}

func (c *Context) AppBounds() image.Rectangle {
	return c.app.bounds()
}

func (c *Context) SetVisible(widget Widget, visible bool) {
	widgetState := widget.widgetState()
	if widgetState.hidden == !visible {
		return
	}
	widgetState.hidden = !visible
	if !visible {
		c.blur(widget)
	}
	_ = traverseWidget(widget, func(w Widget) error {
		w.widgetState().visibleCacheValid = false
		w.widgetState().visibleCache = false
		return nil
	})
	RequestRebuild(widget)
}

func (c *Context) IsVisible(widget Widget) bool {
	return widget.widgetState().isVisible()
}

func (c *Context) SetEnabled(widget Widget, enabled bool) {
	widgetState := widget.widgetState()
	if widgetState.disabled == !enabled {
		return
	}
	widgetState.disabled = !enabled
	if !enabled {
		c.blur(widget)
	}
	_ = traverseWidget(widget, func(w Widget) error {
		w.widgetState().enabledCacheValid = false
		w.widgetState().enabledCache = false
		return nil
	})
	RequestRebuild(widget)
}

func (c *Context) IsEnabled(widget Widget) bool {
	return widget.widgetState().isEnabled()
}

func (c *Context) SetFocused(widget Widget, focused bool) {
	if focused {
		c.focus(widget)
	} else {
		c.blur(widget)
	}
}

func (c *Context) resolveFocusedWidget(widget Widget) Widget {
	origWidget := widget
	visited := map[Widget]struct{}{}
	for {
		if !c.canHaveFocus(widget.widgetState()) {
			return nil
		}
		if widget.widgetState().focusDelegate == nil {
			return widget
		}
		if _, ok := visited[widget]; ok {
			panic(fmt.Sprintf("guigui: infinite focus delegation loop: %T", origWidget))
		}
		visited[widget] = struct{}{}
		widget = widget.widgetState().focusDelegate
	}
}

func (c *Context) focus(widget Widget) {
	ws := c.resolveFocusedWidget(widget)
	c.app.focusWidget(ws)
}

func (c *Context) blur(widget Widget) {
	if c.app.focusedWidget == nil {
		return
	}

	widgetState := widget.widgetState()
	if !widgetState.isInTree(c.app.buildCount) {
		return
	}
	_ = traverseWidget(widget, func(w Widget) error {
		if !areWidgetsSame(c.app.focusedWidget, w) {
			return nil
		}
		for ; w != nil && w.widgetState() != nil; w = w.widgetState().parent {
			if ws := c.resolveFocusedWidget(w); ws != nil && !areWidgetsSame(ws, c.app.focusedWidget) {
				c.app.focusWidget(ws)
				break
			}
		}
		return skipTraverse
	})
}

func (c *Context) canHaveFocus(widgetState *widgetState) bool {
	return widgetState.isInTree(c.app.buildCount) && widgetState.isVisible() && widgetState.isEnabled()
}

func (c *Context) IsFocused(widget Widget) bool {
	return c.canHaveFocus(widget.widgetState()) && areWidgetsSame(c.app.focusedWidget, widget)
}

func (c *Context) IsFocusedOrHasFocusedChild(widget Widget) bool {
	if c.inBuild {
		panic("guigui: IsFocusedOrHasFocusedChild cannot be called in Build")
	}

	if len(widget.widgetState().children) == 0 {
		return areWidgetsSame(c.app.focusedWidget, widget)
	}

	w := c.app.focusedWidget
	if w == nil {
		return false
	}
	for {
		widgetState := widget.widgetState()
		if areWidgetsSame(w, widget) {
			return widgetState.isInTree(c.app.buildCount) && widgetState.isVisible()
		}
		if w.widgetState().parent == nil {
			break
		}
		w = w.widgetState().parent
	}
	return false
}

func (c *Context) Opacity(widget Widget) float64 {
	return widget.widgetState().opacity()
}

func (c *Context) SetOpacity(widget Widget, opacity float64) {
	opacity = min(max(opacity, 0), 1)
	widgetState := widget.widgetState()
	if widgetState.transparency == 1-opacity {
		return
	}
	widgetState.transparency = 1 - opacity
	RequestRebuild(widget)
}

// EnvSource provides information about the origin of an [Context.Env] call.
type EnvSource struct {
	// Origin is the widget that originally called [Context.Env].
	Origin Widget

	// Child is the direct child of the current widget in the walk path.
	// Child is nil when the current widget is the Origin itself.
	Child Widget
}

// Env returns an environment value for the given key by walking up the widget tree.
// It calls [Widget.Env] on the given widget first. If the second return value is false,
// it tries the parent widget, repeating recursively up to the root widget.
func (c *Context) Env(widget Widget, key EnvKey) (any, bool) {
	c.envSource.Origin = widget
	c.envSource.Child = nil

	for w := widget; w != nil; w = w.widgetState().parent {
		if v, ok := w.Env(c, key, &c.envSource); ok {
			return v, true
		}
		c.envSource.Child = w
	}
	return nil, false
}

func (c *Context) Passthrough(widget Widget) bool {
	return widget.widgetState().isPassthrough()
}

func (c *Context) SetPassthrough(widget Widget, passthrough bool) {
	widgetState := widget.widgetState()
	if widgetState.passthrough == passthrough {
		return
	}
	widgetState.passthrough = passthrough
	_ = traverseWidget(widget, func(w Widget) error {
		w.widgetState().passthroughCacheValid = false
		w.widgetState().passthroughCache = false
		return nil
	})
	RequestRebuild(widget)
}

func (c *Context) bringToFrontLayer(widget Widget) {
	widgetState := widget.widgetState()
	// If the widget is already in the front layer, do nothing.
	if widgetState.layer != 0 && widgetState.layer == c.frontLayer {
		return
	}
	// Increment the front layer so that the next layer is always on top.
	c.frontLayer++
	widgetState.layer = c.frontLayer
	_ = traverseWidget(widget, func(w Widget) error {
		w.widgetState().actualLayerPlus1Cache = 0
		return nil
	})
	RequestRebuild(widget)
}

func (c *Context) visibleBounds(state *widgetState) image.Rectangle {
	if state.hasVisibleBoundsCache {
		return state.visibleBoundsCache
	}

	b := state.bounds
	l := state.actualLayer()
	for parent := state.parent; parent != nil; parent = parent.widgetState().parent {
		if parent.widgetState().actualLayer() != l {
			state.hasVisibleBoundsCache = true
			state.visibleBoundsCache = b
			return b
		}
		if parent.widgetState().clipChildren {
			b = b.Intersect(c.visibleBounds(parent.widgetState()))
			break
		}
	}
	state.hasVisibleBoundsCache = true
	state.visibleBoundsCache = b
	return b
}

// SetClipChildren sets whether the children on the same layer are clipped by the widget's bounds.
// The default value is false.
//
// If the child widget is on a different layer from the parent, it is not clipped.
// Note that a widget layer can be controlled by [LayerWidget].
func (c *Context) SetClipChildren(widget Widget, clip bool) {
	widget.widgetState().clipChildren = clip
}

func (c *Context) SetWindowTitle(title string) {
	ebiten.SetWindowTitle(title)
}

// SetWindowSize sets the window size.
func (c *Context) SetWindowSize(width, height int) {
	ebiten.SetWindowSize(width, height)
}

// SetWindowSizeLimits sets the size limits of the window.
// A negative value indicates the size is not limited.
func (c *Context) SetWindowSizeLimits(minw, minh, maxw, maxh int) {
	ebiten.SetWindowSizeLimits(minw, minh, maxw, maxh)
}

func (c *Context) isDefaultProxyMethodCalled() bool {
	return c.defaultProxyMethodCalled
}

func (c *Context) resetDefaultProxyMethodCalled() {
	c.defaultProxyMethodCalled = false
}

func (c *Context) setDefaultProxyMethodCalledFlag() {
	c.defaultProxyMethodCalled = true
}

func (c *Context) isDefaultTickMethodCalled() bool {
	return c.defaultTickMethodCalled
}

func (c *Context) resetDefaultTickMethodCalled() {
	c.defaultTickMethodCalled = false
}

func (c *Context) setDefaultTickMethodCalledFlag() {
	c.defaultTickMethodCalled = true
}

// DelegateFocus delegates the focus to another widget.
func (c *Context) DelegateFocus(widget Widget, delegate Widget) {
	widget.widgetState().focusDelegate = delegate
}
