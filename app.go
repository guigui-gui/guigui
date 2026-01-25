// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type debugMode struct {
	showRenderingRegions bool
	showBuildLogs        bool
	showInputLogs        bool
	deviceScale          float64
}

var theDebugMode debugMode

func init() {
	for _, token := range strings.Split(os.Getenv("GUIGUI_DEBUG"), ",") {
		switch {
		case token == "showrenderingregions":
			theDebugMode.showRenderingRegions = true
		case token == "showbuildlogs":
			theDebugMode.showBuildLogs = true
		case token == "showinputlogs":
			theDebugMode.showInputLogs = true
		case strings.HasPrefix(token, "devicescale="):
			f, err := strconv.ParseFloat(token[len("devicescale="):], 64)
			if err != nil {
				slog.Error(err.Error())
			}
			theDebugMode.deviceScale = f
		case token == "":
		default:
			slog.Warn("unknown debug option", "option", token)
		}
	}
}

type invalidatedRegionsForDebugItem struct {
	region image.Rectangle
	time   int
}

func invalidatedRegionForDebugMaxTime() int {
	return ebiten.TPS() / 5
}

type widgetAndLayer struct {
	widget Widget
	layer  int64
}

type requiredPhases int

const (
	requiredPhasesBuildAndLayout = iota
	// TODO: Use this when appropriated.
	requiredPhasesLayout
	requiredPhasesNone
)

func (r requiredPhases) addBuild() requiredPhases {
	return requiredPhasesBuildAndLayout
}

func (r requiredPhases) addLayout() requiredPhases {
	switch r {
	case requiredPhasesBuildAndLayout:
		return requiredPhasesBuildAndLayout
	case requiredPhasesLayout:
		return requiredPhasesLayout
	case requiredPhasesNone:
		return requiredPhasesLayout
	}
	return r
}

func (r requiredPhases) requiresBuild() bool {
	return r == requiredPhasesBuildAndLayout
}

func (r requiredPhases) requiresLayout() bool {
	return r == requiredPhasesBuildAndLayout || r == requiredPhasesLayout
}

type app struct {
	root           Widget
	context        Context
	visitedLayers  map[int64]struct{}
	layers         []int64
	buildCount     int64
	requiredPhases requiredPhases

	// maybeHitWidgets are widgets and their layer values at the cursor position.
	// maybeHitWidgets are ordered by descending layer values.
	//
	// Layer values are fixed values just after a tree construction, so they are not changed during buildWidgets.
	//
	// maybeHitWidgets includes all the widgets regardless of their Visibility and PassThrough states.
	maybeHitWidgets []widgetAndLayer

	redrawRequestedRegions           redrawRequests
	redrawAndRebuildRequestedRegions redrawRequests
	regionsToDraw                    image.Rectangle

	invalidatedRegionsForDebug []invalidatedRegionsForDebugItem

	screenWidth  float64
	screenHeight float64
	deviceScale  float64

	lastScreenWidth    float64
	lastScreenHeight   float64
	lastCursorPosition image.Point

	focusedWidget Widget

	offscreen   *ebiten.Image
	debugScreen *ebiten.Image
}

var theApp app

type RunOptions struct {
	Title          string
	WindowSize     image.Point
	WindowMinSize  image.Point
	WindowMaxSize  image.Point
	WindowFloating bool
	AppScale       float64

	RunGameOptions *ebiten.RunGameOptions
}

func Run(root Widget, options *RunOptions) error {
	return RunWithCustomFunc(root, options, ebiten.RunGameWithOptions)
}

func RunWithCustomFunc(root Widget, options *RunOptions, f func(game ebiten.Game, options *ebiten.RunGameOptions) error) error {
	if options == nil {
		options = &RunOptions{}
	}

	ebiten.SetWindowTitle(options.Title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenClearedEveryFrame(false)
	if options.WindowSize.X > 0 && options.WindowSize.Y > 0 {
		ebiten.SetWindowSize(options.WindowSize.X, options.WindowSize.Y)
	}
	minW := -1
	minH := -1
	maxW := -1
	maxH := -1
	if options.WindowMinSize.X > 0 {
		minW = options.WindowMinSize.X
	}
	if options.WindowMinSize.Y > 0 {
		minH = options.WindowMinSize.Y
	}
	if options.WindowMaxSize.X > 0 {
		maxW = options.WindowMaxSize.X
	}
	if options.WindowMaxSize.Y > 0 {
		maxH = options.WindowMaxSize.Y
	}
	ebiten.SetWindowSizeLimits(minW, minH, maxW, maxH)
	ebiten.SetWindowFloating(options.WindowFloating)

	a := &theApp
	a.root = root
	a.deviceScale = deviceScaleFactor()
	a.root.widgetState().root = true
	a.context.app = a
	if options.AppScale > 0 {
		a.context.appScaleMinus1 = options.AppScale - 1
	}

	var eop ebiten.RunGameOptions
	if options.RunGameOptions != nil {
		eop = *options.RunGameOptions
	}
	// Prefer SRGB for consistent result.
	if eop.ColorSpace == ebiten.ColorSpaceDefault {
		eop.ColorSpace = ebiten.ColorSpaceSRGB
	}

	return f(a, &eop)
}

func deviceScaleFactor() float64 {
	if theDebugMode.deviceScale != 0 {
		return theDebugMode.deviceScale
	}
	// Calling ebiten.Monitor() seems pretty expensive. Do not call this often.
	// TODO: Ebitengine should be fixed.
	return ebiten.Monitor().DeviceScaleFactor()
}

func (a *app) bounds() image.Rectangle {
	return image.Rect(0, 0, int(math.Ceil(a.screenWidth)), int(math.Ceil(a.screenHeight)))
}

func (a *app) focusWidget(widget Widget) {
	if areWidgetsSame(a.focusedWidget, widget) {
		return
	}
	if a.focusedWidget != nil {
		RequestRebuild(a.focusedWidget)
		DispatchEvent(a.focusedWidget, widgetEventFocusChanged, false)
	}
	a.focusedWidget = widget
	if a.focusedWidget != nil {
		RequestRebuild(a.focusedWidget)
		DispatchEvent(a.focusedWidget, widgetEventFocusChanged, true)
	}

	// Redraw the entire screen, as any widgets can be affected by the focus change (#283).
	// requestRedrawReasonFocus also requests rebuilding a tree.
	a.requestRedraw(a.bounds(), requestRedrawReasonFocus, nil)
}

func (a *app) updateEventDispatchStates() Widget {
	var dispatchedWidget Widget
	_ = traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()
		if widgetState.eventDispatched {
			if dispatchedWidget == nil {
				dispatchedWidget = widget
			}
			widgetState.eventDispatched = false
		}
		return nil
	})
	return dispatchedWidget
}

func (a *app) updateRedrawRequestedRegionsByWidgets() {
	_ = traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()
		if widgetState.rebuildRequested || widgetState.redrawRequested {
			if vb := a.context.visibleBounds(widgetState); !vb.Empty() {
				var reason requestRedrawReason
				if widgetState.rebuildRequested {
					reason = requestRedrawReasonRebuildWidget
				} else {
					reason = requestRedrawReasonRedrawWidget
				}
				a.requestRedrawWidget(widget, reason)
			}
		}
		widgetState.rebuildRequested = false
		widgetState.rebuildRequestedAt = ""
		widgetState.redrawRequested = false
		widgetState.redrawRequestedAt = ""
		return nil
	})
}

func (a *app) Update() error {
	var layoutChangedInUpdate bool

	if a.focusedWidget == nil {
		a.focusWidget(a.root)
	}

	if s := deviceScaleFactor(); a.deviceScale != s {
		a.deviceScale = s
		a.requestRedraw(a.bounds(), requestRedrawReasonScreenDeviceScale, nil)
	}

	rootState := a.root.widgetState()
	rootState.bounds = a.bounds()

	// Call the first buildWidgets.
	a.redrawRequestedRegions.reset()
	a.redrawAndRebuildRequestedRegions.reset()
	if a.requiredPhases.requiresBuild() {
		a.context.inBuild = true
		if err := a.buildWidgets(); err != nil {
			return err
		}
		a.context.inBuild = false
	}
	if a.requiredPhases.requiresLayout() {
		a.layoutWidgets()
		layoutChangedInUpdate = true
	}

	a.updateHitWidgets(a.requiredPhases.requiresLayout())

	// Handle user inputs.
	// TODO: Handle this in Ebitengine's HandleInput in the future (hajimehoshi/ebiten#1704)
	var inputHandledWidget Widget
	if r := a.handleInputWidget(handleInputTypePointing); r.widget != nil {
		if !r.aborted {
			inputHandledWidget = r.widget
		}
		if theDebugMode.showInputLogs {
			slog.Info("pointing input handled", "widget", fmt.Sprintf("%T", r.widget), "aborted", r.aborted)
		}
	}
	if r := a.handleInputWidget(handleInputTypeButton); r.widget != nil {
		if !r.aborted {
			inputHandledWidget = r.widget
		}
		if theDebugMode.showInputLogs {
			slog.Info("keyboard input handled", "widget", fmt.Sprintf("%T", r.widget), "aborted", r.aborted)
		}
	}

	dispatchedWidget := a.updateEventDispatchStates()
	a.updateRedrawRequestedRegionsByWidgets()
	a.requiredPhases = requiredPhasesNone
	if dispatchedWidget != nil {
		a.requiredPhases = a.requiredPhases.addBuild()
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: event dispatched", "widget", fmt.Sprintf("%T", dispatchedWidget))
		}
	}
	if !a.redrawAndRebuildRequestedRegions.empty() {
		a.requiredPhases = a.requiredPhases.addBuild()
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: region redraw requested", "region", a.redrawAndRebuildRequestedRegions)
		}
	}
	if inputHandledWidget != nil {
		a.requiredPhases = a.requiredPhases.addBuild()
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: input handled", "widget", fmt.Sprintf("%T", inputHandledWidget))
		}
	}

	a.regionsToDraw = a.redrawRequestedRegions.union(a.regionsToDraw)
	a.regionsToDraw = a.redrawAndRebuildRequestedRegions.union(a.regionsToDraw)

	// Call the second buildWidgets to construct the widget tree again to reflect the latest state.
	a.redrawRequestedRegions.reset()
	a.redrawAndRebuildRequestedRegions.reset()
	if a.requiredPhases.requiresBuild() {
		a.context.inBuild = true
		if err := a.buildWidgets(); err != nil {
			return err
		}
		a.context.inBuild = false
	}
	if a.requiredPhases.requiresLayout() {
		a.layoutWidgets()
		layoutChangedInUpdate = true
	}

	a.updateHitWidgets(a.requiredPhases.requiresLayout())

	if !a.cursorShape() {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}

	// Tick
	if err := a.tickWidgets(a.root); err != nil {
		return err
	}

	// Invalidate the engire screen if the screen size is changed.
	var screenInvalidated bool
	if a.lastScreenWidth != a.screenWidth {
		screenInvalidated = true
		a.lastScreenWidth = a.screenWidth
	}
	if a.lastScreenHeight != a.screenHeight {
		screenInvalidated = true
		a.lastScreenHeight = a.screenHeight
	}
	if screenInvalidated {
		a.requestRedraw(a.bounds(), requestRedrawReasonScreenSize, nil)
	} else if layoutChangedInUpdate {
		// Invalidate regions if a widget's children state is changed.
		// A widget's bounds might be changed in Widget.Layout, so do this after building and layouting.
		a.requestRedrawIfTreeChanged(a.root)
	}
	// The previous widget state affects requestRedrawIfTreeChanged.
	// After its call, reset the previous widget state.
	a.resetPrevWidgets(a.root)

	dispatchedWidget = a.updateEventDispatchStates()
	a.updateRedrawRequestedRegionsByWidgets()
	a.requiredPhases = requiredPhasesNone
	if dispatchedWidget != nil {
		a.requiredPhases = a.requiredPhases.addBuild()
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: event dispatched", "widget", fmt.Sprintf("%T", dispatchedWidget))
		}
	}
	if !a.redrawAndRebuildRequestedRegions.empty() {
		a.requiredPhases = a.requiredPhases.addBuild()
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: region redraw requested", "region", a.redrawAndRebuildRequestedRegions)
		}
	}

	a.regionsToDraw = a.redrawRequestedRegions.union(a.regionsToDraw)
	a.regionsToDraw = a.redrawAndRebuildRequestedRegions.union(a.regionsToDraw)

	if theDebugMode.showRenderingRegions {
		// Update the regions in the reversed order to remove items.
		for idx := len(a.invalidatedRegionsForDebug) - 1; idx >= 0; idx-- {
			if a.invalidatedRegionsForDebug[idx].time > 0 {
				a.invalidatedRegionsForDebug[idx].time--
			} else {
				a.invalidatedRegionsForDebug = slices.Delete(a.invalidatedRegionsForDebug, idx, idx+1)
			}
		}

		if !a.regionsToDraw.Empty() {
			idx := slices.IndexFunc(a.invalidatedRegionsForDebug, func(i invalidatedRegionsForDebugItem) bool {
				return i.region.Eq(a.regionsToDraw)
			})
			if idx < 0 {
				a.invalidatedRegionsForDebug = append(a.invalidatedRegionsForDebug, invalidatedRegionsForDebugItem{
					region: a.regionsToDraw,
					time:   invalidatedRegionForDebugMaxTime(),
				})
			} else {
				a.invalidatedRegionsForDebug[idx].time = invalidatedRegionForDebugMaxTime()
			}
		}
	}

	return nil
}

func (a *app) Draw(screen *ebiten.Image) {
	origScreen := screen
	if theDebugMode.showRenderingRegions {
		// As the screen is not cleered every frame, create offscreen here to keep the previous contents.
		if a.offscreen != nil {
			if a.offscreen.Bounds().Dx() != screen.Bounds().Dx() || a.offscreen.Bounds().Dy() != screen.Bounds().Dy() {
				a.offscreen.Deallocate()
				a.offscreen = nil
			}
		}
		if a.offscreen == nil {
			a.offscreen = ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
		}
		screen = a.offscreen
	}
	a.drawWidget(screen)
	if origScreen != screen {
		op := &ebiten.DrawImageOptions{}
		op.Blend = ebiten.BlendCopy
		origScreen.DrawImage(a.offscreen, op)
		a.drawDebugIfNeeded(origScreen)
	}
	a.regionsToDraw = image.Rectangle{}
}

func (a *app) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("guigui: game.Layout should never be called")
}

func (a *app) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	s := a.deviceScale
	a.screenWidth = outsideWidth * s
	a.screenHeight = outsideHeight * s
	return a.screenWidth, a.screenHeight
}

func (a *app) requestRedraw(region image.Rectangle, reason requestRedrawReason, widget Widget) {
	switch reason {
	case requestRedrawReasonRedrawWidget, requestRedrawReasonLayout:
		a.redrawRequestedRegions.add(region, reason, widget)
	default:
		a.redrawAndRebuildRequestedRegions.add(region, reason, widget)
	}
}

func (a *app) requestRedrawWidget(widget Widget, reason requestRedrawReason) {
	widgetState := widget.widgetState()
	a.requestRedraw(a.context.visibleBounds(widgetState), reason, widget)
	for _, child := range widgetState.children {
		a.requestRedrawIfDifferentParentLayer(child, reason)
	}
}

func (a *app) requestRedrawIfDifferentParentLayer(widget Widget, reason requestRedrawReason) {
	widgetState := widget.widgetState()
	if widgetState.inDifferentLayerFromParent() {
		a.requestRedrawWidget(widget, reason)
		return
	}
	for _, child := range widgetState.children {
		a.requestRedrawIfDifferentParentLayer(child, reason)
	}
}

func (a *app) buildWidgets() error {
	a.buildCount++

	a.root.widgetState().builtAt = a.buildCount

	// Clear event handlers to prevent unexpected handlings.
	// An event handler is often a closure capturing variables, and this might cause unexpected behaviors.
	_ = traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()
		clear(widgetState.eventHandlers)
		widgetState.focusDelegation = nil

		widgetState.layerPlus1Cache = 0
		widgetState.visibleCache = false
		widgetState.visibleCacheValid = false
		widgetState.enabledCache = false
		widgetState.enabledCacheValid = false
		widgetState.passThroughCacheValid = false
		widgetState.passThroughCache = false
		// Do not reset bounds an zs here, as they are used to determine whether redraw is needed.
		return nil
	})

	var adder ChildAdder
	if err := traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()
		widgetState.children = slices.Delete(widgetState.children, 0, len(widgetState.children))
		adder.app = a
		adder.widget = widget
		if err := widget.Build(&a.context, &adder); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (a *app) layoutWidgets() {
	clear(a.visitedLayers)
	if a.visitedLayers == nil {
		a.visitedLayers = map[int64]struct{}{}
	}

	var layouter ChildLayouter
	_ = traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()
		widgetState.hasVisibleBoundsCache = false
		widgetState.visibleBoundsCache = image.Rectangle{}

		// Reset child layouts.
		for _, child := range widgetState.children {
			child.widgetState().bounds = image.Rectangle{}
		}

		// Call Layout.
		bounds := widgetBoundsFromWidget(&a.context, widget)
		widget.Layout(&a.context, bounds, &layouter)

		a.visitedLayers[widgetState.actualLayer()] = struct{}{}

		return nil
	})

	a.layers = slices.Delete(a.layers, 0, len(a.layers))
	a.layers = slices.AppendSeq(a.layers, maps.Keys(a.visitedLayers))
	slices.Sort(a.layers)
}

func (a *app) updateHitWidgets(layoutChanged bool) {
	pt := image.Pt(ebiten.CursorPosition())
	if !layoutChanged && pt == a.lastCursorPosition {
		return
	}
	a.lastCursorPosition = pt

	a.maybeHitWidgets = slices.Delete(a.maybeHitWidgets, 0, len(a.maybeHitWidgets))
	a.maybeHitWidgets = a.appendWidgetsAt(a.maybeHitWidgets, pt, a.root, true)
	slices.SortStableFunc(a.maybeHitWidgets, func(a, b widgetAndLayer) int {
		return cmp.Compare(b.layer, a.layer)
	})
}

type handleInputType int

const (
	handleInputTypePointing handleInputType = iota
	handleInputTypeButton
)

func (a *app) handleInputWidget(typ handleInputType) HandleInputResult {
	for i := len(a.layers) - 1; i >= 0; i-- {
		layer := a.layers[i]
		if r := a.doHandleInputWidget(typ, a.root, layer, a.context.IsFocused(a.root)); r.shouldRaise() {
			return r
		}
	}
	return HandleInputResult{}
}

func (a *app) doHandleInputWidget(typ handleInputType, widget Widget, layerToHandle int64, ancestorFocused bool) HandleInputResult {
	widgetState := widget.widgetState()
	if widgetState.isPassThrough() {
		return HandleInputResult{}
	}

	// Avoid (*Context).IsVisible and (*Context).IsEnabled for performance.
	// These check parent widget states unnecessarily.

	if widgetState.hidden {
		return HandleInputResult{}
	}

	if widgetState.disabled {
		return HandleInputResult{}
	}

	if typ == handleInputTypeButton && !a.context.IsFocusedOrHasFocusedChild(widget) && !ancestorFocused {
		return HandleInputResult{}
	}

	// Iterate the children in the reverse order of rendering.
	focused := a.context.IsFocused(widget)
	for i := len(widgetState.children) - 1; i >= 0; i-- {
		child := widgetState.children[i]
		if r := a.doHandleInputWidget(typ, child, layerToHandle, ancestorFocused || focused); r.shouldRaise() {
			return r
		}
	}

	if layerToHandle != widgetState.actualLayer() {
		return HandleInputResult{}
	}

	bounds := widgetBoundsFromWidget(&a.context, widget)

	switch typ {
	case handleInputTypePointing:
		return widget.HandlePointingInput(&a.context, bounds)
	case handleInputTypeButton:
		return widget.HandleButtonInput(&a.context, bounds)
	default:
		panic(fmt.Sprintf("guigui: unknown handleInputType: %d", typ))
	}
}

func (a *app) cursorShape() bool {
	var layer int64
	for _, wl := range a.maybeHitWidgets {
		if layer > wl.layer {
			return false
		}

		widgetState := wl.widget.widgetState()
		if !widgetState.isVisible() {
			continue
		}
		if widgetState.isPassThrough() {
			continue
		}
		if isProxyWidget(&a.context, wl.widget) {
			continue
		}
		if !widgetState.isEnabled() {
			return false
		}
		bounds := widgetBoundsFromWidget(&a.context, wl.widget)
		shape, ok := wl.widget.CursorShape(&a.context, bounds)
		if !ok {
			layer = wl.layer
			continue
		}
		ebiten.SetCursorShape(shape)
		return true
	}
	return false
}

func (a *app) tickWidgets(widget Widget) error {
	bounds := widgetBoundsFromWidget(&a.context, widget)
	if err := widget.Tick(&a.context, bounds); err != nil {
		return err
	}

	for _, child := range widget.widgetState().children {
		if err := a.tickWidgets(child); err != nil {
			return err
		}
	}

	return nil
}

func (a *app) requestRedrawIfTreeChanged(widget Widget) {
	widgetState := widget.widgetState()
	// If the children and/or children's bounds are changed, request redraw.
	if !widgetState.prev.equals(&a.context, widgetState.children) {
		a.requestRedraw(a.context.visibleBounds(widgetState), requestRedrawReasonLayout, nil)

		widgetState.prev.requestRedraw(a)

		// If the widget is a clipping widget, all the children are included in the visible bounds.
		if !widgetState.clipChildren {
			for _, child := range widgetState.children {
				a.requestRedraw(a.context.visibleBounds(child.widgetState()), requestRedrawReasonLayout, nil)
			}
		}
	}
	for _, child := range widgetState.children {
		a.requestRedrawIfTreeChanged(child)
	}
}

func (a *app) resetPrevWidgets(widget Widget) {
	widgetState := widget.widgetState()
	// Reset the states.
	widgetState.prev.reset()
	for _, child := range widgetState.children {
		widgetState.prev.append(&a.context, child)
	}
	for _, child := range widgetState.children {
		a.resetPrevWidgets(child)
	}
}

func (a *app) drawWidget(screen *ebiten.Image) {
	if a.regionsToDraw.Empty() {
		return
	}
	dst := screen.SubImage(a.regionsToDraw).(*ebiten.Image)
	for _, layer := range a.layers {
		a.doDrawWidget(dst, a.root, layer)
	}
}

func (a *app) doDrawWidget(dst *ebiten.Image, widget Widget, layerToRender int64) {
	// Do not skip this even when visible bounds are empty.
	// A child widget might have a different layer value and different visible bounds.

	widgetState := widget.widgetState()
	if widgetState.hidden {
		return
	}
	if dst.Bounds().Empty() {
		return
	}
	opacity := widgetState.opacity()
	if opacity == 0 {
		return
	}
	vb := a.context.visibleBounds(widgetState)
	var copiedDst *ebiten.Image
	renderCurrent := layerToRender == widgetState.actualLayer() && !dst.Bounds().Intersect(vb).Empty()
	if renderCurrent {
		if opacity < 1 {
			// Keep the current destination image to draw it with the opacity later.
			copiedDst = widgetState.ensureOffscreen(dst.Bounds())
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(dst.Bounds().Min.X), float64(dst.Bounds().Min.Y))
			op.Blend = ebiten.BlendCopy
			copiedDst.DrawImage(dst, op)
		}
		widgetBounds := widgetBoundsFromWidget(&a.context, widget)
		widget.Draw(&a.context, widgetBounds, dst.SubImage(vb).(*ebiten.Image))
	}

	for _, child := range widgetState.children {
		a.doDrawWidget(dst, child, layerToRender)
	}

	if renderCurrent {
		if opacity < 1 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(dst.Bounds().Min.X), float64(dst.Bounds().Min.Y))
			op.ColorScale.ScaleAlpha(1 - float32(opacity))
			dst.DrawImage(copiedDst, op)
		}
	}
}

func (a *app) drawDebugIfNeeded(screen *ebiten.Image) {
	if !theDebugMode.showRenderingRegions {
		return
	}

	if a.debugScreen != nil {
		if a.debugScreen.Bounds().Dx() != screen.Bounds().Dx() || a.debugScreen.Bounds().Dy() != screen.Bounds().Dy() {
			a.debugScreen.Deallocate()
			a.debugScreen = nil
		}
	}
	if a.debugScreen == nil {
		a.debugScreen = ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
	}

	a.debugScreen.Clear()
	for _, item := range a.invalidatedRegionsForDebug {
		if alpha := float64(item.time) / float64(invalidatedRegionForDebugMaxTime()); alpha > 0 {
			w := float32(4 * a.context.Scale())
			clr := color.NRGBA{R: 0xff, G: 0x4b, B: 0x00, A: uint8(alpha * 255)}
			vector.StrokeRect(a.debugScreen, float32(item.region.Min.X)+w/2, float32(item.region.Min.Y)+w/2, float32(item.region.Dx())-w, float32(item.region.Dy())-w, w, clr, false)
		}
	}
	screen.DrawImage(a.debugScreen, nil)
}

func (a *app) isWidgetHitAtCursor(widget Widget) bool {
	widgetState := widget.widgetState()
	if !widgetState.isInTree(a.buildCount) {
		return false
	}
	if !widgetState.isVisible() {
		return false
	}
	if widgetState.isPassThrough() {
		return false
	}
	if isProxyWidget(&a.context, widget) {
		return false
	}

	layer := widgetState.actualLayer()

	// hitWidgets are ordered by descending layer values.
	// Always use a fixed set hitWidgets, as the tree might be dynamically changed during buildWidgets.
	for _, wl := range a.maybeHitWidgets {
		if wl.widget.widgetState() == widgetState {
			return true
		}

		l1 := wl.layer
		if l1 < layer {
			// The same layer value no longer exists.
			return false
		}

		if l1 == layer {
			continue
		}

		if !wl.widget.widgetState().isVisible() {
			continue
		}
		if wl.widget.widgetState().isPassThrough() {
			continue
		}
		if isProxyWidget(&a.context, wl.widget) {
			continue
		}

		// w overlaps widget at point.
		return false
	}
	return false
}

func (a *app) appendWidgetsAt(widgets []widgetAndLayer, point image.Point, widget Widget, parentHit bool) []widgetAndLayer {
	widgetState := widget.widgetState()
	var hit bool
	if parentHit || widgetState.inDifferentLayerFromParent() {
		hit = point.In(a.context.visibleBounds(widgetState))
	}

	children := widgetState.children
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		widgets = a.appendWidgetsAt(widgets, point, child, hit)
	}

	if !hit {
		return widgets
	}

	widgets = append(widgets, widgetAndLayer{
		widget: widget,
		layer:  widgetState.actualLayer(),
	})
	return widgets
}
