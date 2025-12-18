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
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui/internal/colormode"
	"github.com/guigui-gui/guigui/internal/locale"
)

type ColorMode int

var envLocales []language.Tag

func init() {
	if locales := os.Getenv("GUIGUI_LOCALES"); locales != "" {
		for _, tag := range strings.Split(os.Getenv("GUIGUI_LOCALES"), ",") {
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

const (
	ColorModeLight ColorMode = iota
	ColorModeDark
)

type Context struct {
	app     *app
	inBuild bool

	appScaleMinus1             float64
	colorMode                  ColorMode
	colorModeSet               bool
	cachedDefaultColorMode     colormode.ColorMode
	cachedDefaultColorModeTime time.Time
	defaultColorWarnOnce       sync.Once
	locales                    []language.Tag
	allLocales                 []language.Tag
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

func (c *Context) ColorMode() ColorMode {
	if c.colorModeSet {
		return c.colorMode
	}
	return c.autoColorMode()
}

func (c *Context) SetColorMode(mode ColorMode) {
	if c.colorModeSet && mode == c.colorMode {
		return
	}

	c.colorMode = mode
	c.colorModeSet = true
	c.app.requestRedraw(c.app.bounds(), requestRedrawReasonColorMode, nil)
}

func (c *Context) UseAutoColorMode() {
	if !c.colorModeSet {
		return
	}
	c.colorModeSet = false
	c.app.requestRedraw(c.app.bounds(), requestRedrawReasonColorMode, nil)
}

func (c *Context) IsAutoColorModeUsed() bool {
	return !c.colorModeSet
}

func (c *Context) autoColorMode() ColorMode {
	// TODO: Consider the system color mode.
	switch mode := os.Getenv("GUIGUI_COLOR_MODE"); mode {
	case "light":
		return ColorModeLight
	case "dark":
		return ColorModeDark
	case "":
		if time.Since(c.cachedDefaultColorModeTime) >= time.Second {
			m := colormode.SystemColorMode()
			if c.cachedDefaultColorMode != m {
				c.app.requestRedraw(c.app.bounds(), requestRedrawReasonColorMode, nil)
			}
			c.cachedDefaultColorMode = m
			c.cachedDefaultColorModeTime = time.Now()
		}
		switch c.cachedDefaultColorMode {
		case colormode.Light:
			return ColorModeLight
		case colormode.Dark:
			return ColorModeDark
		}
	default:
		c.defaultColorWarnOnce.Do(func() {
			slog.Warn(fmt.Sprintf("invalid GUIGUI_COLOR_MODE: %s", mode))
		})
	}

	return ColorModeLight
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
	RequestRedraw(widget)
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
	RequestRedraw(widget)
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
	for {
		if !c.canHaveFocus(widget.widgetState()) {
			return nil
		}
		if widget.widgetState().focusDelegation == nil {
			return widget
		}
		widget = widget.widgetState().focusDelegation
	}
}

func (c *Context) focus(widget Widget) {
	ws := c.resolveFocusedWidget(widget)
	if areWidgetsSame(c.app.focusedWidget, ws) {
		return
	}

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
		if c.app.focusedWidget.widgetState() != w.widgetState() {
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
	RequestRedraw(widget)
}

func (c *Context) SetCustomDraw(widget Widget, customDraw CustomDrawFunc) {
	widget.widgetState().customDraw = customDraw
}

func (c *Context) Model(widget Widget, key any) any {
	for w := widget; w != nil; w = w.widgetState().parent {
		if v := w.Model(key); v != nil {
			return v
		}
	}
	return nil
}

func (c *Context) PassThrough(widget Widget) bool {
	return widget.widgetState().passThrough
}

func (c *Context) SetPassThrough(widget Widget, passThrough bool) {
	widgetState := widget.widgetState()
	if widgetState.passThrough == passThrough {
		return
	}
	widgetState.passThrough = passThrough
	RequestRedraw(widget)
}

func (c *Context) ZDelta(widget Widget) int {
	return widget.widgetState().zDelta
}

func (c *Context) SetZDelta(widget Widget, zDelta int) {
	if zDelta < 0 {
		panic("guigui: ZDelta must be non-negative")
	}
	widgetState := widget.widgetState()
	if widgetState.zDelta == zDelta {
		return
	}
	widgetState.zDelta = zDelta
	_ = traverseWidget(widget, func(w Widget) error {
		w.widgetState().zPlus1Cache = 0
		return nil
	})
	RequestRedraw(widget)
}

func (c *Context) visibleBounds(state *widgetState) image.Rectangle {
	if state.hasVisibleBoundsCache {
		return state.visibleBoundsCache
	}

	parent := state.parent
	if parent == nil {
		b := c.app.bounds()
		state.hasVisibleBoundsCache = true
		state.visibleBoundsCache = b
		return b
	}
	if state.zDelta != 0 {
		b := state.bounds
		state.hasVisibleBoundsCache = true
		state.visibleBoundsCache = b
		return b
	}
	if state.float {
		b := state.bounds
		for parent := state.parent; parent != nil; parent = parent.widgetState().parent {
			if parent.widgetState().container {
				parentVB := c.visibleBounds(parent.widgetState())
				if !parentVB.Empty() {
					b = b.Intersect(parentVB)
				}
				break
			}
		}
		state.hasVisibleBoundsCache = true
		state.visibleBoundsCache = b
		return b
	}

	var b image.Rectangle
	parentVB := c.visibleBounds(parent.widgetState())
	if !parentVB.Empty() {
		b = parentVB.Intersect(state.bounds)
	}
	state.hasVisibleBoundsCache = true
	state.visibleBoundsCache = b
	return b
}

func (c *Context) SetContainer(widget Widget, container bool) {
	widget.widgetState().container = container
}

func (c *Context) SetFloat(widget Widget, float bool) {
	widget.widgetState().float = float
}

func (c *Context) DelegateFocus(from Widget, to Widget) {
	if areWidgetsSame(from.widgetState().focusDelegation, to) {
		return
	}
	from.widgetState().focusDelegation = to
}

func (c *Context) SetWindowTitle(title string) {
	ebiten.SetWindowTitle(title)
}
