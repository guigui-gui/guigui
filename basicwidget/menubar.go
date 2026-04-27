// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package basicwidget

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

var (
	menubarEventItemSelected guigui.EventKey = guigui.GenerateEventKey()
)

// MenubarItem is a single entry in a [Menubar]. Each entry has a title text;
// the popup menu shown when the title is clicked is configured separately via
// [Menubar.PopupMenuAt].
type MenubarItem struct {
	Text     string
	Disabled bool
}

// Menubar is a horizontal row of title texts. Clicking a title shows the
// associated popup menu. While any popup is open, moving the cursor onto a
// different title automatically switches to that title's popup. Clicking the
// already-open title closes its popup.
type Menubar[T comparable] struct {
	guigui.DefaultWidget

	items  []MenubarItem
	titles guigui.WidgetSlice[*menubarTitle[T]]
	popups guigui.WidgetSlice[*PopupMenu[T]]

	layoutItems []guigui.LinearLayoutItem
	titleBounds []image.Rectangle

	// openIndexPlus1 is the index of the currently open popup, plus one.
	// Zero means no popup is open. The plus-one offset keeps the zero value
	// of the widget meaningful.
	openIndexPlus1 int

	// lastAppliedOpenIndexPlus1 mirrors the open index that was last propagated
	// to the popup widgets. Build only calls SetOpen on transitions, so closed
	// popups are not re-toggled (and their internal RequestRebuild is not fired)
	// on every Build.
	lastAppliedOpenIndexPlus1 int

	onItemSelectedHandlers []func(context *guigui.Context, itemIndex int)
	onCloseHandlers        []func(context *guigui.Context, reason PopupCloseReason)
}

// SetItems sets the menubar's items. After this call, the popup menu for each
// item can be configured via [Menubar.PopupMenuAt].
func (m *Menubar[T]) SetItems(items []MenubarItem) {
	m.items = adjustSliceSize(m.items, len(items))
	copy(m.items, items)
	m.titles.SetLen(len(items))
	m.popups.SetLen(len(items))
	m.onItemSelectedHandlers = adjustSliceSize(m.onItemSelectedHandlers, len(items))
	m.onCloseHandlers = adjustSliceSize(m.onCloseHandlers, len(items))
	if m.openIndexPlus1 > len(m.items) {
		m.openIndexPlus1 = 0
	}
}

// PopupMenuAt returns the popup menu associated with the i-th item.
// [Menubar.SetItems] must be called first so that the popup at i exists.
//
// Items shown in the popup are configured via [PopupMenu.SetItems]. A title
// whose popup has no items is automatically disabled.
func (m *Menubar[T]) PopupMenuAt(index int) *PopupMenu[T] {
	return m.popups.At(index)
}

// OnItemSelected sets the event handler that is invoked when a popup menu
// item is selected. menuIndex identifies the title and itemIndex identifies
// the selected item within that title's popup.
func (m *Menubar[T]) OnItemSelected(f func(context *guigui.Context, menuIndex, itemIndex int)) {
	guigui.SetEventHandler(m, menubarEventItemSelected, f)
}

// WriteStateKey implements [guigui.Widget.WriteStateKey].
func (m *Menubar[T]) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteInt(m.openIndexPlus1)
}

func (m *Menubar[T]) requestOpen(index int) {
	if index < 0 {
		m.openIndexPlus1 = 0
		return
	}
	if index >= len(m.items) {
		return
	}
	if m.items[index].Disabled || len(m.popups.At(index).items) == 0 {
		return
	}
	m.openIndexPlus1 = index + 1
}

// Build implements [guigui.Widget.Build].
func (m *Menubar[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if m.openIndexPlus1 < 0 || m.openIndexPlus1 > len(m.items) {
		m.openIndexPlus1 = 0
	}
	openIndex := m.openIndexPlus1 - 1

	for i := range m.items {
		title := m.titles.At(i)
		title.menubar = m
		title.index = i
		title.setText(m.items[i].Text)
		adder.AddWidget(title)
		context.SetEnabled(title, !m.items[i].Disabled && len(m.popups.At(i).items) > 0)
	}

	for i := range m.items {
		popup := m.popups.At(i)
		popup.setModal(false)

		if m.onItemSelectedHandlers[i] == nil {
			idx := i
			m.onItemSelectedHandlers[idx] = func(context *guigui.Context, itemIndex int) {
				if m.openIndexPlus1 == idx+1 {
					m.openIndexPlus1 = 0
				}
				guigui.DispatchEvent(m, menubarEventItemSelected, idx, itemIndex)
			}
		}
		popup.OnItemSelected(m.onItemSelectedHandlers[i])

		if m.onCloseHandlers[i] == nil {
			idx := i
			m.onCloseHandlers[idx] = func(context *guigui.Context, reason PopupCloseReason) {
				if m.openIndexPlus1 == idx+1 {
					m.openIndexPlus1 = 0
				}
			}
		}
		popup.OnClose(m.onCloseHandlers[i])
	}

	// Apply transitions only when the open index actually changes, so closed
	// popups don't get their toClose flag toggled on every Build.
	if m.lastAppliedOpenIndexPlus1 != m.openIndexPlus1 {
		if prev := m.lastAppliedOpenIndexPlus1 - 1; prev >= 0 && prev < len(m.items) {
			m.popups.At(prev).SetOpen(false)
		}
		if openIndex >= 0 && openIndex < len(m.items) {
			m.popups.At(openIndex).SetOpen(true)
		}
		m.lastAppliedOpenIndexPlus1 = m.openIndexPlus1
	}

	// Keep popup widgets in the tree while they are open or animating, so
	// their hide animation can progress.
	for i := range m.items {
		if m.popups.At(i).IsOpen() {
			adder.AddWidget(m.popups.At(i))
		}
	}

	return nil
}

func (m *Menubar[T]) layout(context *guigui.Context) guigui.LinearLayout {
	m.layoutItems = slices.Delete(m.layoutItems, 0, len(m.layoutItems))
	for i := range m.titles.Len() {
		m.layoutItems = append(m.layoutItems, guigui.LinearLayoutItem{
			Widget: m.titles.At(i),
		})
	}
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     m.layoutItems,
		Padding: guigui.Padding{
			Start: RoundedCornerRadius(context),
			End:   RoundedCornerRadius(context),
		},
	}
}

// Layout implements [guigui.Widget.Layout].
func (m *Menubar[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	bounds := widgetBounds.Bounds()

	layout := m.layout(context)
	layout.LayoutWidgets(context, bounds, layouter)
	m.titleBounds = layout.AppendItemBounds(m.titleBounds[:0], context, bounds)

	for i := range m.popups.Len() {
		popup := m.popups.At(i)
		if !popup.IsOpen() {
			continue
		}
		// Exclude the menubar bounds from close-by-clicking-outside, so a
		// click on a title doesn't briefly close the popup before re-opening.
		popup.setCloseByClickingOutsideExcludedRect(bounds)
		size := popup.Measure(context, guigui.Constraints{})
		tb := m.titleBounds[i]
		// Attach the popup to the bottom of the title's highlighted background
		// rectangle, not the title's full bounds, so the popup looks flush
		// with the highlight pill.
		bb := m.titles.At(i).backgroundBounds(context, tb)
		popupBounds := image.Rect(bb.Min.X, bb.Max.Y, bb.Min.X+size.X, bb.Max.Y+size.Y)
		layouter.LayoutWidget(popup, popupBounds)
	}
}

// Measure implements [guigui.Widget.Measure].
func (m *Menubar[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return m.layout(context).Measure(context, constraints)
}

// menubarTitle is the per-item title widget rendered inside a [Menubar].
type menubarTitle[T comparable] struct {
	guigui.DefaultWidget

	menubar *Menubar[T]
	index   int
	text    Text

	layoutItems        []guigui.LinearLayoutItem
	wrapperLayoutItems []guigui.LinearLayoutItem
}

func (t *menubarTitle[T]) setText(value string) {
	t.text.SetValue(value)
}

func (t *menubarTitle[T]) WriteStateKey(w *guigui.StateKeyWriter) {
	w.WriteInt(t.index)
}

func (t *menubarTitle[T]) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.text)
	t.text.SetHorizontalAlign(HorizontalAlignCenter)
	t.text.SetVerticalAlign(VerticalAlignMiddle)

	if t.isOpen() && context.IsEnabled(t) {
		t.text.SetColor(ListItemColorTypeHighlighted.TextColor(context))
	} else {
		t.text.SetColor(nil)
	}
	return nil
}

func (t *menubarTitle[T]) layout(context *guigui.Context) guigui.LinearLayout {
	t.layoutItems = slices.Delete(t.layoutItems, 0, len(t.layoutItems))
	t.layoutItems = append(t.layoutItems, guigui.LinearLayoutItem{
		Widget: &t.text,
		Size:   guigui.FlexibleSize(1),
	})
	inner := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.layoutItems,
		Padding:   ListItemTextPadding(context),
	}
	t.wrapperLayoutItems = slices.Delete(t.wrapperLayoutItems, 0, len(t.wrapperLayoutItems))
	t.wrapperLayoutItems = append(t.wrapperLayoutItems, guigui.LinearLayoutItem{
		Layout: inner,
		Size:   guigui.FixedSize(UnitSize(context)),
	})
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     t.wrapperLayoutItems,
	}
}

func (t *menubarTitle[T]) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	t.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *menubarTitle[T]) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.layout(context).Measure(context, constraints)
}

func (t *menubarTitle[T]) isOpen() bool {
	return t.menubar != nil && t.menubar.openIndexPlus1 == t.index+1
}

func (t *menubarTitle[T]) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !context.IsEnabled(t) {
		return guigui.HandleInputResult{}
	}
	if !widgetBounds.IsHitAtCursor() {
		return guigui.HandleInputResult{}
	}

	// Click: toggle this title's popup.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if t.isOpen() {
			t.menubar.requestOpen(-1)
		} else {
			t.menubar.requestOpen(t.index)
		}
		return guigui.HandleInputByWidget(t)
	}

	// Hover-to-switch: only switch while another popup is already open.
	if t.menubar.openIndexPlus1 != 0 && !t.isOpen() {
		t.menubar.requestOpen(t.index)
	}
	return guigui.HandleInputResult{}
}

func (t *menubarTitle[T]) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if context.IsEnabled(t) && widgetBounds.IsHitAtCursor() {
		return ebiten.CursorShapePointer, true
	}
	return 0, false
}

// backgroundBounds returns the rectangle for the highlighted background drawn
// behind the title text. The height matches a list item's highlighted
// background (text height plus top/bottom of [ListItemTextPadding]) and the
// rectangle is vertically centered within the title's full bounds.
func (t *menubarTitle[T]) backgroundBounds(context *guigui.Context, bounds image.Rectangle) image.Rectangle {
	p := ListItemTextPadding(context)
	h := t.text.Measure(context, guigui.Constraints{}).Y + p.Top + p.Bottom
	y := bounds.Min.Y + (bounds.Dy()-h)/2
	return image.Rect(bounds.Min.X, y, bounds.Max.X, y+h)
}

func (t *menubarTitle[T]) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	dst.Fill(basicwidgetdraw.BackgroundColorFromSemanticColor(context.ColorMode(), basicwidgetdraw.SemanticColorBase))

	if !context.IsEnabled(t) {
		return
	}
	if !t.isOpen() {
		return
	}
	clr := draw.Color2(context.ColorMode(), draw.SemanticColorAccent, 0.6, 0.475)
	basicwidgetdraw.DrawRoundedRect(context, dst, t.backgroundBounds(context, widgetBounds.Bounds()), clr, RoundedCornerRadius(context))
}
