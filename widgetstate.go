// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

import (
	"errors"
	"fmt"
	"image"
	"maps"
	"reflect"
	"runtime"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

type bounds3D struct {
	// Use bounds here. Visual bounds don't work to detect tree changes.

	bounds        image.Rectangle
	visibleBounds image.Rectangle
	float         bool
	layer         int64
	visible       bool // For hit testing.
	passThrough   bool // For hit testing.
}

func bounds3DFromWidget(context *Context, widget Widget) (bounds3D, bool) {
	ws := widget.widgetState()
	b := ws.bounds
	if b.Empty() {
		return bounds3D{}, false
	}
	vb := context.visibleBounds(ws)
	if vb.Empty() {
		return bounds3D{}, false
	}
	return bounds3D{
		bounds:        b,
		visibleBounds: vb,
		float:         ws.floating,
		layer:         ws.actualLayer(),
		visible:       ws.isVisible(),
		passThrough:   ws.passThrough,
	}, true
}

type widgetsAndBounds struct {
	bounds3Ds       map[*widgetState]bounds3D
	currentBounds3D map[*widgetState]bounds3D
}

func (w *widgetsAndBounds) reset() {
	clear(w.bounds3Ds)
}

func (w *widgetsAndBounds) append(context *Context, widget Widget) {
	if w.bounds3Ds == nil {
		w.bounds3Ds = map[*widgetState]bounds3D{}
	}
	b, ok := bounds3DFromWidget(context, widget)
	if !ok {
		return
	}
	w.bounds3Ds[widget.widgetState()] = b
}

func (w *widgetsAndBounds) equals(context *Context, currentWidgets []Widget) bool {
	if w.currentBounds3D == nil {
		w.currentBounds3D = map[*widgetState]bounds3D{}
	} else {
		clear(w.currentBounds3D)
	}
	for _, widget := range currentWidgets {
		b, ok := bounds3DFromWidget(context, widget)
		if !ok {
			continue
		}
		w.currentBounds3D[widget.widgetState()] = b
	}
	return maps.Equal(w.bounds3Ds, w.currentBounds3D)
}

func (w *widgetsAndBounds) redrawIfNeeded(app *app) {
	for widgetState, bounds3D := range w.bounds3Ds {
		if widgetState.inDifferentLayerFromParent() || bounds3D.float {
			app.requestRedraw(bounds3D.visibleBounds, requestRedrawReasonLayout, nil)
			requestRedraw(widgetState)
		}
	}
}

type CustomDrawFunc func(dst, widgetImage *ebiten.Image, op *ebiten.DrawImageOptions)

type widgetState struct {
	root    bool
	builtAt int64

	bounds image.Rectangle

	parent   Widget
	children []Widget
	prev     widgetsAndBounds

	hidden          bool
	disabled        bool
	passThrough     bool
	layer           int64
	transparency    float64
	customDraw      CustomDrawFunc
	eventHandlers   map[EventKey]any
	tmpArgs         []reflect.Value
	eventDispatched bool
	floatingClip    bool
	floating        bool
	focusDelegation Widget

	layerPlus1Cache       int64
	visibleCache          bool
	visibleCacheValid     bool
	enabledCache          bool
	enabledCacheValid     bool
	passThroughCache      bool
	passThroughCacheValid bool

	offscreen *ebiten.Image

	rebuildRequested   bool
	rebuildRequestedAt string
	redrawRequested    bool
	redrawRequestedAt  string

	hasVisibleBoundsCache bool
	visibleBoundsCache    image.Rectangle

	widgetBounds_ WidgetBounds

	isProxyCacheValid bool
	isProxyCache      bool

	_ noCopy
}

func (w *widgetState) isInTree(now int64) bool {
	return w.builtAt == now
}

func (w *widgetState) isVisible() bool {
	if w.visibleCacheValid {
		return w.visibleCache
	}
	w.visibleCacheValid = true
	if w.hidden {
		w.visibleCache = false
	} else if w.parent != nil {
		w.visibleCache = w.parent.widgetState().isVisible()
	} else {
		w.visibleCache = true
	}
	return w.visibleCache
}

func (w *widgetState) isEnabled() bool {
	if w.enabledCacheValid {
		return w.enabledCache
	}
	w.enabledCacheValid = true
	if w.disabled {
		w.enabledCache = false
	} else if w.parent != nil {
		w.enabledCache = w.parent.widgetState().isEnabled()
	} else {
		w.enabledCache = true
	}
	return w.enabledCache
}

func (w *widgetState) isPassThrough() bool {
	if w.passThroughCacheValid {
		return w.passThroughCache
	}
	w.passThroughCacheValid = true
	if w.passThrough {
		w.passThroughCache = true
	} else if w.parent != nil {
		w.passThroughCache = w.parent.widgetState().isPassThrough()
	} else {
		w.passThroughCache = false
	}
	return w.passThroughCache
}

func (w *widgetState) opacity() float64 {
	return 1 - w.transparency
}

func (w *widgetState) ensureOffscreen(bounds image.Rectangle) *ebiten.Image {
	if w.offscreen != nil {
		if !bounds.In(w.offscreen.Bounds()) {
			w.offscreen.Deallocate()
			w.offscreen = nil
		}
	}
	if w.offscreen == nil {
		w.offscreen = ebiten.NewImageWithOptions(bounds, nil)
	}
	return w.offscreen.SubImage(bounds).(*ebiten.Image)
}
func (w *widgetState) actualLayer() int64 {
	if w.layerPlus1Cache != 0 {
		return w.layerPlus1Cache - 1
	}
	var layer int64
	if w.layer != 0 {
		layer = w.layer
	} else if w.parent == nil {
		layer = w.layer
	} else {
		layer = w.parent.widgetState().actualLayer()
	}
	w.layerPlus1Cache = layer + 1
	return layer
}

func (w *widgetState) inDifferentLayerFromParent() bool {
	if w.parent == nil {
		return false
	}
	return w.actualLayer() != w.parent.widgetState().actualLayer()
}

var (
	dummyImage = ebiten.NewImage(1, 1)
)

// isProxyWidget returns true if the widget is a proxy.
// A proxy widget is a widget whose Draw, HandlePointingInput, and CursorShape are the default implementation.
// A proxy widget mainly manages its children and doesn't handle pointing input and drawing.
// A proxy widget is ignored for cursor hit tests.
func isProxyWidget(context *Context, widget Widget) bool {
	if widget.widgetState().isProxyCacheValid {
		return widget.widgetState().isProxyCache
	}

	// Do not use widgetBoundsFromWidget returning a cached WidgetBounds.
	// Disable the hit test, or isProxyWidget will be recursively called at HandlePointingInput.
	wb := WidgetBounds{
		widget:      widget,
		context:     context,
		hitDisabled: true,
	}

	isProxy := true
	// Actually invoke HandlePointingInput and Draw to check if they are the default implementation.
	// TODO: Is this safe?
	context.resetDefaultMethodCalled()
	widget.HandlePointingInput(context, &wb)
	if !context.isDefaultMethodCalled() {
		isProxy = false
	}
	context.resetDefaultMethodCalled()
	widget.Draw(context, &wb, dummyImage)
	if !context.isDefaultMethodCalled() {
		isProxy = false
	}
	context.resetDefaultMethodCalled()
	widget.CursorShape(context, &wb)
	if !context.isDefaultMethodCalled() {
		isProxy = false
	}

	widget.widgetState().isProxyCacheValid = true
	widget.widgetState().isProxyCache = isProxy
	return isProxy
}

func widgetBoundsFromWidget(context *Context, widget Widget) *WidgetBounds {
	wb := &widget.widgetState().widgetBounds_
	wb.widget = widget
	wb.context = context
	return wb
}

var skipTraverse = errors.New("skip traverse")

func traverseWidget(widget Widget, f func(widget Widget) error) error {
	if err := f(widget); err != nil {
		return err
	}
	for _, child := range widget.widgetState().children {
		if err := traverseWidget(child, f); err != nil {
			return err
		}
	}
	return nil
}

func RequestRebuild(widget Widget) {
	requestRebuild(widget.widgetState())
}

func requestRebuild(widgetState *widgetState) {
	widgetState.rebuildRequested = true
	if theDebugMode.showRenderingRegions {
		if _, file, line, ok := runtime.Caller(2); ok {
			widgetState.rebuildRequestedAt = fmt.Sprintf("%s:%d", file, line)
		}
	}
}

func RequestRedraw(widget Widget) {
	requestRedraw(widget.widgetState())
}

func requestRedraw(widgetState *widgetState) {
	widgetState.redrawRequested = true
	if theDebugMode.showRenderingRegions {
		if _, file, line, ok := runtime.Caller(2); ok {
			widgetState.redrawRequestedAt = fmt.Sprintf("%s:%d", file, line)
		}
	}
}

func SetEventHandler(widget Widget, eventKey EventKey, handler any) {
	widgetState := widget.widgetState()
	if widgetState.eventHandlers == nil {
		widgetState.eventHandlers = map[EventKey]any{}
	}
	widgetState.eventHandlers[eventKey] = handler
}

func DispatchEvent(widget Widget, eventKey EventKey, args ...any) {
	widgetState := widget.widgetState()
	hanlder, ok := widgetState.eventHandlers[eventKey]
	if !ok {
		return
	}
	f := reflect.ValueOf(hanlder)
	widgetState.tmpArgs = slices.Delete(widgetState.tmpArgs, 0, len(widgetState.tmpArgs))
	widgetState.tmpArgs = append(widgetState.tmpArgs, reflect.ValueOf(&theApp.context))
	for _, arg := range args {
		widgetState.tmpArgs = append(widgetState.tmpArgs, reflect.ValueOf(arg))
	}
	f.Call(widgetState.tmpArgs)
	widgetState.tmpArgs = slices.Delete(widgetState.tmpArgs, 0, len(widgetState.tmpArgs))
	widgetState.eventDispatched = true

	RequestRebuild(widget)
}

// noCopy is a struct to warn that the struct should not be copied.
//
// For details, see https://go.dev/issues/8005#issuecomment-190753527
type noCopy struct {
}

func (n *noCopy) Lock() {
}

func (n *noCopy) Unlock() {
}
