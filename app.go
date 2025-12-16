// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package guigui

import (
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

type widgetAndZ struct {
	widget Widget
	z      int
}

type requiredPhases int

const (
	requiredPhasesBuildAndLayout = iota
	// TODO: Use this when appropriated.
	requiredPhasesLayout
	requiredPhasesNone
)

func (r requiredPhases) requiresBuild() bool {
	return r == requiredPhasesBuildAndLayout
}

func (r requiredPhases) requiresLayout() bool {
	return r == requiredPhasesBuildAndLayout || r == requiredPhasesLayout
}

type app struct {
	root           Widget
	context        Context
	visitedZs      map[int]struct{}
	zs             []int
	buildCount     int64
	requiredPhases requiredPhases

	// maybeHitWidgets are widgets and their z values at the cursor position.
	// maybeHitWidgets are ordered by descending z values.
	//
	// Z values are fixed values just after a tree construction, so they are not changed during buildWidgets.
	//
	// maybeHitWidgets includes all the widgets regardless of their Visibility and PassThrough states.
	maybeHitWidgets []widgetAndZ

	invalidatedRegions image.Rectangle

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
		a.focusedWidget.OnFocusChanged(&a.context, false)
	}
	a.focusedWidget = widget
	if a.focusedWidget != nil {
		a.focusedWidget.OnFocusChanged(&a.context, true)
	}
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

func (a *app) updateInvalidatedRegions() {
	_ = traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()
		if !widgetState.redrawRequested {
			return nil
		}
		if vb := a.context.visibleBounds(widgetState); !vb.Empty() {
			if theDebugMode.showRenderingRegions {
				slog.Info("request redrawing", "requester", fmt.Sprintf("%T", widget), "at", widgetState.redrawRequestedAt, "region", vb)
			}
			a.requestRedrawWidget(widget)
		}
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
		a.requestRedraw(a.bounds())
	}

	rootState := a.root.widgetState()
	rootState.bounds = a.bounds()

	// Call the first buildWidgets.
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
	a.updateInvalidatedRegions()
	a.requiredPhases = requiredPhasesNone
	if dispatchedWidget != nil {
		a.requiredPhases = requiredPhasesBuildAndLayout
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: event dispatched", "widget", fmt.Sprintf("%T", dispatchedWidget))
		}
	} else if inputHandledWidget != nil {
		a.requiredPhases = requiredPhasesBuildAndLayout
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: input handled", "widget", fmt.Sprintf("%T", inputHandledWidget))
		}
	} else if !a.invalidatedRegions.Empty() {
		a.requiredPhases = requiredPhasesBuildAndLayout
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: region invalidated", "region", a.invalidatedRegions)
		}
	}

	// Call the second buildWidgets to construct the widget tree again to reflect the latest state.
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
		a.requestRedraw(a.bounds())
	} else if layoutChangedInUpdate {
		// Invalidate regions if a widget's children state is changed.
		// A widget's bounds might be changed in Widget.Layout, so do this after building and layouting.
		a.requestRedrawIfTreeChanged(a.root)
	}

	a.resetPrevWidgets(a.root)

	dispatchedWidget = a.updateEventDispatchStates()
	a.updateInvalidatedRegions()
	a.requiredPhases = requiredPhasesNone
	if dispatchedWidget != nil {
		a.requiredPhases = requiredPhasesBuildAndLayout
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: event dispatched", "widget", fmt.Sprintf("%T", dispatchedWidget))
		}
	} else if !a.invalidatedRegions.Empty() {
		a.requiredPhases = requiredPhasesBuildAndLayout
		if theDebugMode.showBuildLogs {
			slog.Info("rebuilding tree next time: region invalidated", "region", a.invalidatedRegions)
		}
	}

	if theDebugMode.showRenderingRegions {
		// Update the regions in the reversed order to remove items.
		for idx := len(a.invalidatedRegionsForDebug) - 1; idx >= 0; idx-- {
			if a.invalidatedRegionsForDebug[idx].time > 0 {
				a.invalidatedRegionsForDebug[idx].time--
			} else {
				a.invalidatedRegionsForDebug = slices.Delete(a.invalidatedRegionsForDebug, idx, idx+1)
			}
		}

		if !a.invalidatedRegions.Empty() {
			idx := slices.IndexFunc(a.invalidatedRegionsForDebug, func(i invalidatedRegionsForDebugItem) bool {
				return i.region.Eq(a.invalidatedRegions)
			})
			if idx < 0 {
				a.invalidatedRegionsForDebug = append(a.invalidatedRegionsForDebug, invalidatedRegionsForDebugItem{
					region: a.invalidatedRegions,
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
	a.invalidatedRegions = image.Rectangle{}
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

func (a *app) requestRedraw(region image.Rectangle) {
	a.invalidatedRegions = a.invalidatedRegions.Union(region)
}

func (a *app) requestRedrawWidget(widget Widget) {
	widgetState := widget.widgetState()
	a.requestRedraw(a.context.visibleBounds(widgetState))
	for _, child := range widgetState.children {
		a.requestRedrawIfDifferentParentZ(child)
	}
}

func (a *app) requestRedrawIfDifferentParentZ(widget Widget) {
	widgetState := widget.widgetState()
	if widgetState.zDelta != 0 {
		a.requestRedrawWidget(widget)
		return
	}
	for _, child := range widgetState.children {
		a.requestRedrawIfDifferentParentZ(child)
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
		widgetState.zPlus1Cache = 0
		widgetState.visibleCache = false
		widgetState.visibleCacheValid = false
		widgetState.enabledCache = false
		widgetState.enabledCacheValid = false
		// Do not reset bounds an zs here, as they are used to determine whether redraw is needed.
		return nil
	})

	var adder ChildAdder
	if err := traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()
		widgetState.hasVisibleBoundsCache = false
		widgetState.visibleBoundsCache = image.Rectangle{}
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
	clear(a.visitedZs)
	if a.visitedZs == nil {
		a.visitedZs = map[int]struct{}{}
	}

	var layouter ChildLayouter
	_ = traverseWidget(a.root, func(widget Widget) error {
		widgetState := widget.widgetState()

		// Reset child layouts.
		for _, child := range widgetState.children {
			child.widgetState().bounds = image.Rectangle{}
		}

		// Call Layout.
		bounds := widgetBoundsFromWidget(&a.context, widgetState)
		widget.Layout(&a.context, bounds, &layouter)

		a.visitedZs[widgetState.z()] = struct{}{}

		return nil
	})

	a.zs = slices.Delete(a.zs, 0, len(a.zs))
	a.zs = slices.AppendSeq(a.zs, maps.Keys(a.visitedZs))
	slices.Sort(a.zs)
}

func (a *app) updateHitWidgets(layoutChanged bool) {
	pt := image.Pt(ebiten.CursorPosition())
	if !layoutChanged && pt == a.lastCursorPosition {
		return
	}
	a.lastCursorPosition = pt

	a.maybeHitWidgets = slices.Delete(a.maybeHitWidgets, 0, len(a.maybeHitWidgets))
	a.maybeHitWidgets = a.appendWidgetsAt(a.maybeHitWidgets, pt, a.root, true)
	slices.SortStableFunc(a.maybeHitWidgets, func(a, b widgetAndZ) int {
		return b.z - a.z
	})
}

type handleInputType int

const (
	handleInputTypePointing handleInputType = iota
	handleInputTypeButton
)

func (a *app) handleInputWidget(typ handleInputType) HandleInputResult {
	for i := len(a.zs) - 1; i >= 0; i-- {
		z := a.zs[i]
		if r := a.doHandleInputWidget(typ, a.root, z, a.context.IsFocused(a.root)); r.shouldRaise() {
			return r
		}
	}
	return HandleInputResult{}
}

func (a *app) doHandleInputWidget(typ handleInputType, widget Widget, zToHandle int, ancestorFocused bool) HandleInputResult {
	widgetState := widget.widgetState()
	if widgetState.passThrough {
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
		if !ancestorFocused {
			return HandleInputResult{}
		}
	}

	// Iterate the children in the reverse order of rendering.
	focused := a.context.IsFocused(widget)
	for i := len(widgetState.children) - 1; i >= 0; i-- {
		child := widgetState.children[i]
		if r := a.doHandleInputWidget(typ, child, zToHandle, ancestorFocused || focused); r.shouldRaise() {
			return r
		}
	}

	if zToHandle != widgetState.z() {
		return HandleInputResult{}
	}

	bounds := widgetBoundsFromWidget(&a.context, widgetState)

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
	for _, wz := range a.maybeHitWidgets {
		widgetState := wz.widget.widgetState()
		if widgetState.passThrough {
			continue
		}
		if !widgetState.isVisible() {
			continue
		}
		if !widgetState.isEnabled() {
			return false
		}
		bounds := widgetBoundsFromWidget(&a.context, widgetState)
		shape, ok := wz.widget.CursorShape(&a.context, bounds)
		if !ok {
			continue
		}
		ebiten.SetCursorShape(shape)
		return true
	}
	return false
}

func (a *app) tickWidgets(widget Widget) error {
	widgetState := widget.widgetState()
	bounds := widgetBoundsFromWidget(&a.context, widgetState)
	if err := widget.Tick(&a.context, bounds); err != nil {
		return err
	}

	for _, child := range widgetState.children {
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
		a.requestRedraw(a.context.visibleBounds(widgetState))

		// Widgets with different Z from their parent's Z (e.g. popups) are outside of widget, so redraw the regions explicitly.
		// The float property is similar.
		widgetState.prev.redrawIfNeeded(a)
		for _, child := range widgetState.children {
			if child.widgetState().zDelta != 0 || child.widgetState().float {
				a.requestRedraw(a.context.visibleBounds(child.widgetState()))
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
	if a.invalidatedRegions.Empty() {
		return
	}
	dst := screen.SubImage(a.invalidatedRegions).(*ebiten.Image)
	for _, z := range a.zs {
		a.doDrawWidget(dst, a.root, z, false)
		a.doDrawWidget(dst, a.root, z, true)
	}
}

func (a *app) doDrawWidget(dst *ebiten.Image, widget Widget, zToRender int, float bool) {
	// Do not skip this even when visible bounds are empty.
	// A child widget might have a different Z value and different visible bounds.

	widgetState := widget.widgetState()
	if widgetState.hidden {
		return
	}
	if widgetState.opacity() == 0 {
		return
	}

	customDraw := widgetState.customDraw
	useOffscreen := (widgetState.opacity() < 1 || customDraw != nil) && !dst.Bounds().Empty()

	vb := a.context.visibleBounds(widgetState)
	var origDst *ebiten.Image
	renderCurrent := zToRender == widgetState.z() && !vb.Empty() && widgetState.float == float
	if renderCurrent {
		if useOffscreen {
			origDst = dst
			dst = widgetState.ensureOffscreen(dst.Bounds())
			dst.Clear()
		}
		widgetBounds := widgetBoundsFromWidget(&a.context, widgetState)
		widget.Draw(&a.context, widgetBounds, dst.SubImage(vb).(*ebiten.Image))
	}

	for _, child := range widgetState.children {
		a.doDrawWidget(dst, child, zToRender, float)
	}

	if renderCurrent {
		if useOffscreen {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(dst.Bounds().Min.X), float64(dst.Bounds().Min.Y))
			op.ColorScale.ScaleAlpha(float32(widgetState.opacity()))
			if customDraw != nil {
				customDraw(origDst.SubImage(vb).(*ebiten.Image), dst, op)
			} else {
				origDst.DrawImage(dst, op)
			}
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

func (a *app) isWidgetHitAtCursor(widgetState *widgetState) bool {
	if !widgetState.isInTree(a.buildCount) {
		return false
	}
	if !widgetState.isVisible() {
		return false
	}
	if widgetState.passThrough {
		return false
	}

	z := widgetState.z()

	// hitWidgets are ordered by descending z values.
	// Always use a fixed set hitWidgets, as the tree might be dynamically changed during buildWidgets.
	for _, wz := range a.maybeHitWidgets {
		z1 := wz.z
		if z1 > z {
			// w overlaps widget at point.
			if wz.widget.widgetState().isVisible() && !wz.widget.widgetState().passThrough {
				return false
			}
			continue
		}
		if z1 < z {
			// The same z value no longer exists.
			return false
		}

		if wz.widget.widgetState() == widgetState {
			return true
		}
	}
	return false
}

func (a *app) appendWidgetsAt(widgets []widgetAndZ, point image.Point, widget Widget, parentHit bool) []widgetAndZ {
	widgetState := widget.widgetState()
	var hit bool
	if parentHit || widgetState.zDelta != 0 {
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

	widgets = append(widgets, widgetAndZ{
		widget: widget,
		z:      widgetState.z(),
	})
	return widgets
}
