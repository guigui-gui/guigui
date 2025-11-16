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
	visibleBounds image.Rectangle
	float         bool
	zDelta        int
	visible       bool // For hit testing.
	passThrough   bool // For hit testing.
}

func bounds3DFromWidget(context *Context, widget Widget) (bounds3D, bool) {
	ws := widget.widgetState()
	vb := context.visibleBounds(ws)
	if vb.Empty() {
		return bounds3D{}, false
	}
	return bounds3D{
		visibleBounds: vb,
		float:         ws.float,
		zDelta:        ws.zDelta,
		visible:       ws.isVisible(),
		passThrough:   ws.passThrough,
	}, true
}

type widgetsAndVisibleBounds struct {
	bounds3Ds       map[*widgetState]bounds3D
	currentBounds3D map[*widgetState]bounds3D
}

func (w *widgetsAndVisibleBounds) reset() {
	clear(w.bounds3Ds)
}

func (w *widgetsAndVisibleBounds) append(context *Context, widget Widget) {
	if w.bounds3Ds == nil {
		w.bounds3Ds = map[*widgetState]bounds3D{}
	}
	b, ok := bounds3DFromWidget(context, widget)
	if !ok {
		return
	}
	w.bounds3Ds[widget.widgetState()] = b
}

func (w *widgetsAndVisibleBounds) equals(context *Context, currentWidgets []Widget) bool {
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

func (w *widgetsAndVisibleBounds) redrawIfNeeded(app *app) {
	for widgetState, bounds3D := range w.bounds3Ds {
		if bounds3D.zDelta != 0 || bounds3D.float {
			app.requestRedraw(bounds3D.visibleBounds)
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
	prev     widgetsAndVisibleBounds

	hidden          bool
	disabled        bool
	passThrough     bool
	zDelta          int
	zPlus1Cache     int
	transparency    float64
	customDraw      CustomDrawFunc
	eventHandlers   map[string]any
	tmpArgs         []reflect.Value
	eventDispatched bool
	container       bool
	float           bool
	focusDelegation *widgetState

	offscreen *ebiten.Image

	dirty   bool
	dirtyAt string

	hasVisibleBoundsCache bool
	visibleBoundsCache    image.Rectangle

	widgetBounds_ WidgetBounds

	_ noCopy
}

func (w *widgetState) isInTree(now int64) bool {
	return w.builtAt == now
}

func (w *widgetState) isVisible() bool {
	if w.parent != nil {
		if w.hidden {
			return false
		}
		return w.parent.widgetState().isVisible()
	}
	return !w.hidden
}

func (w *widgetState) isEnabled() bool {
	if w.parent != nil {
		if w.disabled {
			return false
		}
		return w.parent.widgetState().isEnabled()
	}
	return !w.disabled
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

func (w *widgetState) z() int {
	if w.zPlus1Cache != 0 {
		return w.zPlus1Cache - 1
	}
	var z int
	if w.parent == nil {
		z = w.zDelta
	} else {
		z = w.parent.widgetState().z() + w.zDelta
	}
	w.zPlus1Cache = z + 1
	return z
}

func widgetBoundsFromWidget(context *Context, widgetState *widgetState) *WidgetBounds {
	wb := &widgetState.widgetBounds_
	wb.widgetState = widgetState
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

func RequestRedraw(widget Widget) {
	requestRedraw(widget.widgetState())
}

func requestRedraw(widgetState *widgetState) {
	widgetState.dirty = true
	if theDebugMode.showRenderingRegions {
		if _, file, line, ok := runtime.Caller(2); ok {
			widgetState.dirtyAt = fmt.Sprintf("%s:%d", file, line)
		}
	}
}

func RegisterEventHandler(widget Widget, eventName string, handler any) {
	widgetState := widget.widgetState()
	if widgetState.eventHandlers == nil {
		widgetState.eventHandlers = map[string]any{}
	}
	widgetState.eventHandlers[eventName] = handler
}

func IsEventHandlerRegistered(widget Widget, eventName string) bool {
	widgetState := widget.widgetState()
	_, ok := widgetState.eventHandlers[eventName]
	return ok
}

func DispatchEventHandler(widget Widget, eventName string, args ...any) ([]any, bool) {
	return dispatchEventHandler(widget.widgetState(), eventName, args...)
}

func dispatchEventHandler(widgetState *widgetState, eventName string, args ...any) ([]any, bool) {
	hanlder, ok := widgetState.eventHandlers[eventName]
	if !ok {
		return nil, false
	}
	f := reflect.ValueOf(hanlder)
	widgetState.tmpArgs = slices.Delete(widgetState.tmpArgs, 0, len(widgetState.tmpArgs))
	for _, arg := range args {
		widgetState.tmpArgs = append(widgetState.tmpArgs, reflect.ValueOf(arg))
	}
	returns := f.Call(widgetState.tmpArgs)
	widgetState.tmpArgs = slices.Delete(widgetState.tmpArgs, 0, len(widgetState.tmpArgs))
	var v []any
	for _, ret := range returns {
		v = append(v, ret.Interface())
	}
	widgetState.eventDispatched = true
	return v, true
}

const focusChangedEvent = "__focusChanged"

// SetOnFocusChanged registers a handler for focus changed events.
//
// A handler can be dispatched even when the widget is not in the tree.
func SetOnFocusChanged(widget Widget, f func(focused bool)) {
	RegisterEventHandler(widget, focusChangedEvent, f)
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
