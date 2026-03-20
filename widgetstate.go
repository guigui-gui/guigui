// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

import (
	"errors"
	"fmt"
	"image"
	"reflect"
	"runtime"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

type bounds3D struct {
	// Use bounds here. Visual bounds don't work to detect tree changes.

	bounds        image.Rectangle
	visibleBounds image.Rectangle
	layer         int64
	visible       bool // For hit testing.
	passthrough   bool // For hit testing.
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
		layer:         ws.actualLayer(),
		visible:       ws.isVisible(),
		passthrough:   ws.passthrough,
	}, true
}

type widgetsAndBounds struct {
	bounds3Ds       []widgetStateAndBounds
	currentBounds3D []widgetStateAndBounds
}

type widgetStateAndBounds struct {
	widgetState *widgetState
	bounds3D    bounds3D
}

func (w *widgetsAndBounds) reset() {
	w.bounds3Ds = slices.Delete(w.bounds3Ds, 0, len(w.bounds3Ds))
}

func (w *widgetsAndBounds) append(context *Context, widget Widget) {
	b, ok := bounds3DFromWidget(context, widget)
	if !ok {
		return
	}
	w.bounds3Ds = append(w.bounds3Ds, widgetStateAndBounds{
		widgetState: widget.widgetState(),
		bounds3D:    b,
	})
}

func (w *widgetsAndBounds) equals(context *Context, currentWidgets []Widget) bool {
	w.currentBounds3D = slices.Delete(w.currentBounds3D, 0, len(w.currentBounds3D))
	for _, widget := range currentWidgets {
		b, ok := bounds3DFromWidget(context, widget)
		if !ok {
			continue
		}
		w.currentBounds3D = append(w.currentBounds3D, widgetStateAndBounds{
			widgetState: widget.widgetState(),
			bounds3D:    b,
		})
	}
	return slices.Equal(w.bounds3Ds, w.currentBounds3D)
}

func (w *widgetsAndBounds) requestRedraw(app *app) {
	for _, wb := range w.bounds3Ds {
		app.requestRedraw(wb.bounds3D.visibleBounds, requestRedrawReasonLayout, nil)
		requestRedraw(wb.widgetState)
	}
}

type widgetState struct {
	root    bool
	builtAt int64

	bounds image.Rectangle

	parent        Widget
	children      []Widget
	prev          widgetsAndBounds
	focusDelegate Widget

	hidden       bool
	disabled     bool
	passthrough  bool
	layer        int64
	transparency float64

	// eventHandlers is a collection of event handlers.
	// eventHandlers is reset whenever the widget is rebuilt.
	//
	// Use a slice instead of a map for performance.
	// Especially, clearing a map is costly.
	eventHandlers []eventHandler

	tmpArgs         []reflect.Value
	eventDispatched bool
	clipChildren    bool

	actualLayerPlus1Cache int64
	visibleCache          bool
	visibleCacheValid     bool
	enabledCache          bool
	enabledCacheValid     bool
	passthroughCache      bool
	passthroughCacheValid bool

	offscreen *ebiten.Image

	rebuildRequested      bool
	rebuildRequestedAt    string
	redrawReasonOnRebuild requestRedrawReason

	redrawRequested   bool
	redrawRequestedAt string

	hasVisibleBoundsCache bool
	visibleBoundsCache    image.Rectangle

	widgetBounds_ WidgetBounds

	isProxyCacheValid bool
	isProxyCache      bool

	_ noCopy
}

type eventHandler struct {
	key     EventKey
	handler any
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

func (w *widgetState) isPassthrough() bool {
	if w.passthroughCacheValid {
		return w.passthroughCache
	}
	w.passthroughCacheValid = true
	if w.passthrough {
		w.passthroughCache = true
	} else if w.parent != nil {
		w.passthroughCache = w.parent.widgetState().isPassthrough()
	} else {
		w.passthroughCache = false
	}
	return w.passthroughCache
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
	if w.actualLayerPlus1Cache != 0 {
		return w.actualLayerPlus1Cache - 1
	}
	// The layer is determined by the closest ancestor with a non-zero layer.
	// For example, if there are three popups A, B, and C, and B is a child of A.
	// If A's layer is 1, B's layer is 3, and C's layer is 2, then the popups are
	// rendered in the order of A, C, and B, even though B is a child of A.
	var layer int64
	if w.layer != 0 {
		layer = w.layer
	} else if w.parent == nil {
		layer = w.layer
	} else {
		layer = w.parent.widgetState().actualLayer()
	}
	w.actualLayerPlus1Cache = layer + 1
	return layer
}

func (w *widgetState) inDifferentLayerFromParent() bool {
	if w.parent == nil {
		return w.layer != 0
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
	theApp.requestRebuild(widget.widgetState(), requestRedrawReasonRebuildWidget)
}

func (a *app) requestRebuild(widgetState *widgetState, redrawReason requestRedrawReason) {
	if !widgetState.isInTree(a.buildCount) {
		// requestRebuild can be called with a widget that is not in the tree.
		// For example, a popup widget that is not added yet can invoke this when opening it.
		// As the special case, rebuild the root widget.
		widgetState = a.root.widgetState()
	}
	widgetState.rebuildRequested = true
	widgetState.redrawReasonOnRebuild = redrawReason
	a.hasDirtyWidgets = true
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
	theApp.hasDirtyWidgets = true
	if theDebugMode.showRenderingRegions {
		if _, file, line, ok := runtime.Caller(2); ok {
			widgetState.redrawRequestedAt = fmt.Sprintf("%s:%d", file, line)
		}
	}
}

// SetEventHandler registers an event handler for the given event key on the widget.
// At most one handler can be registered per event key on a widget.
// If a handler is already registered for the same key, it is replaced.
// All event handlers are reset before the build phase starts,
// so SetEventHandler must be called during every Build to keep the handler active.
func SetEventHandler(widget Widget, eventKey EventKey, handler any) {
	widgetState := widget.widgetState()
	widgetState.eventHandlers = slices.DeleteFunc(widgetState.eventHandlers, func(h eventHandler) bool {
		return h.key == eventKey
	})
	widgetState.eventHandlers = append(widgetState.eventHandlers, eventHandler{
		key:     eventKey,
		handler: handler,
	})
}

// DispatchEvent invokes the event handler registered for the given event key on the widget.
// The handler must have been set via SetEventHandler during the current build phase,
// as all handlers are reset before each build.
// args are passed to the handler after the *Context argument.
// It returns the handler's return values and true if a handler was found, or nil and false otherwise.
func DispatchEvent(widget Widget, eventKey EventKey, args ...any) ([]any, bool) {
	widgetState := widget.widgetState()
	for _, h := range widgetState.eventHandlers {
		if h.key != eventKey {
			continue
		}
		f := reflect.ValueOf(h.handler)
		widgetState.tmpArgs = slices.Delete(widgetState.tmpArgs, 0, len(widgetState.tmpArgs))
		widgetState.tmpArgs = append(widgetState.tmpArgs, reflect.ValueOf(&theApp.context))
		for _, arg := range args {
			widgetState.tmpArgs = append(widgetState.tmpArgs, reflect.ValueOf(arg))
		}
		results := f.Call(widgetState.tmpArgs)
		widgetState.tmpArgs = slices.Delete(widgetState.tmpArgs, 0, len(widgetState.tmpArgs))
		widgetState.eventDispatched = true
		theApp.hasDirtyWidgets = true
		RequestRebuild(widget)
		if len(results) == 0 {
			return nil, true
		}
		ret := make([]any, len(results))
		for i, r := range results {
			ret[i] = r.Interface()
		}
		return ret, true
	}
	return nil, false
}

var widgetEventFocusChanged EventKey = GenerateEventKey()

// TODO: For focus delegation, create a new function (#340).
func OnFocusChanged(widget Widget, onfocus func(context *Context, focused bool)) {
	SetEventHandler(widget, widgetEventFocusChanged, onfocus)
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
