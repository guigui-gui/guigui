// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

import (
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

var (
	menubarEventNew               = guigui.GenerateEventKey()
	menubarEventOpen              = guigui.GenerateEventKey()
	menubarEventSave              = guigui.GenerateEventKey()
	menubarEventSaveAs            = guigui.GenerateEventKey()
	menubarEventUndo              = guigui.GenerateEventKey()
	menubarEventRedo              = guigui.GenerateEventKey()
	menubarEventCut               = guigui.GenerateEventKey()
	menubarEventCopy              = guigui.GenerateEventKey()
	menubarEventPaste             = guigui.GenerateEventKey()
	menubarEventFind              = guigui.GenerateEventKey()
	menubarEventSelectAll         = guigui.GenerateEventKey()
	menubarEventExtraItemSelected = guigui.GenerateEventKey()
)

// builtinMenuCount is the number of built-in menus, File and Edit, that
// precede the extra menus.
const builtinMenuCount = 2

// ExtraMenu is an application-specific menu shown after the built-in menus.
type ExtraMenu struct {
	// Text is the menu's title in the menubar.
	Text string

	// Items are the menu's popup items. Their values are reported as-is by
	// the handler set by [Menubar.OnExtraItemSelected].
	Items []basicwidget.PopupMenuItem[string]

	// ReservesCheckmarkSpace indents the items so that toggling a checkmark
	// does not shift their texts.
	ReservesCheckmarkSpace bool
}

// Menubar is an editor application's menubar widget. It wraps
// [basicwidget.Menubar] with the File and Edit menus common to the editor
// examples and exposes typed event handlers (OnNew, OnOpen, ...) so that an
// application can register actions without dealing with menu/item indices.
type Menubar struct {
	guigui.DefaultWidget

	menubar basicwidget.Menubar[string]

	canSave    bool
	canUndo    bool
	canRedo    bool
	canCut     bool
	canCopy    bool
	canPaste   bool
	extraMenus []ExtraMenu
}

// SetCanSave enables or disables the Save item.
func (m *Menubar) SetCanSave(b bool) {
	m.canSave = b
}

// SetCanUndo enables or disables the Undo item.
func (m *Menubar) SetCanUndo(b bool) {
	m.canUndo = b
}

// SetCanRedo enables or disables the Redo item.
func (m *Menubar) SetCanRedo(b bool) {
	m.canRedo = b
}

// SetCanCut enables or disables the Cut item.
func (m *Menubar) SetCanCut(b bool) {
	m.canCut = b
}

// SetCanCopy enables or disables the Copy item.
func (m *Menubar) SetCanCopy(b bool) {
	m.canCopy = b
}

// SetCanPaste enables or disables the Paste item.
func (m *Menubar) SetCanPaste(b bool) {
	m.canPaste = b
}

// SetExtraMenus sets the application-specific menus shown after the built-in
// menus.
func (m *Menubar) SetExtraMenus(menus []ExtraMenu) {
	m.extraMenus = slices.Delete(m.extraMenus, 0, len(m.extraMenus))
	m.extraMenus = append(m.extraMenus, menus...)
}

// OnNew sets the event handler invoked when the New item is selected.
func (m *Menubar) OnNew(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventNew, fn)
}

// OnOpen sets the event handler invoked when the Open item is selected.
func (m *Menubar) OnOpen(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventOpen, fn)
}

// OnSave sets the event handler invoked when the Save item is selected.
func (m *Menubar) OnSave(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventSave, fn)
}

// OnSaveAs sets the event handler invoked when the Save As item is selected.
func (m *Menubar) OnSaveAs(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventSaveAs, fn)
}

// OnUndo sets the event handler invoked when the Undo item is selected.
func (m *Menubar) OnUndo(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventUndo, fn)
}

// OnRedo sets the event handler invoked when the Redo item is selected.
func (m *Menubar) OnRedo(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventRedo, fn)
}

// OnCut sets the event handler invoked when the Cut item is selected.
func (m *Menubar) OnCut(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventCut, fn)
}

// OnCopy sets the event handler invoked when the Copy item is selected.
func (m *Menubar) OnCopy(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventCopy, fn)
}

// OnPaste sets the event handler invoked when the Paste item is selected.
func (m *Menubar) OnPaste(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventPaste, fn)
}

// OnFind sets the event handler invoked when the Find item is selected.
func (m *Menubar) OnFind(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventFind, fn)
}

// OnSelectAll sets the event handler invoked when the Select All item is
// selected.
func (m *Menubar) OnSelectAll(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, menubarEventSelectAll, fn)
}

// OnExtraItemSelected sets the event handler invoked when an item of a menu
// set by [Menubar.SetExtraMenus] is selected. value is the selected item's
// value.
func (m *Menubar) OnExtraItemSelected(fn func(context *guigui.Context, value string)) {
	guigui.SetEventHandler(m, menubarEventExtraItemSelected, fn)
}

func (m *Menubar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&m.menubar)

	menubarItems := []basicwidget.MenubarItem{
		{Text: "File"},
		{Text: "Edit"},
	}
	popupItems := [][]basicwidget.PopupMenuItem[string]{
		{
			{Text: "New", Value: "new", KeyText: Hotkey("N")},
			{Text: "Open…", Value: "open", KeyText: Hotkey("O")},
			{Border: true},
			{Text: "Save", Value: "save", KeyText: Hotkey("S"), Disabled: !m.canSave},
			{Text: "Save As…", Value: "saveas"},
		},
		{
			{Text: "Undo", Value: "undo", KeyText: Hotkey("Z"), Disabled: !m.canUndo},
			{Text: "Redo", Value: "redo", KeyText: HotkeyShift("Z"), Disabled: !m.canRedo},
			{Border: true},
			{Text: "Cut", Value: "cut", KeyText: Hotkey("X"), Disabled: !m.canCut},
			{Text: "Copy", Value: "copy", KeyText: Hotkey("C"), Disabled: !m.canCopy},
			{Text: "Paste", Value: "paste", KeyText: Hotkey("V"), Disabled: !m.canPaste},
			{Border: true},
			{Text: "Find…", Value: "find", KeyText: Hotkey("F")},
			{Border: true},
			{Text: "Select All", Value: "selectall", KeyText: Hotkey("A")},
		},
	}
	for _, menu := range m.extraMenus {
		menubarItems = append(menubarItems, basicwidget.MenubarItem{Text: menu.Text})
		popupItems = append(popupItems, menu.Items)
	}

	m.menubar.SetItems(menubarItems)
	for i, items := range popupItems {
		m.menubar.PopupMenuAt(i).SetItems(items)
	}
	for i, menu := range m.extraMenus {
		m.menubar.PopupMenuAt(builtinMenuCount + i).SetReservesCheckmarkSpace(menu.ReservesCheckmarkSpace)
	}

	m.menubar.OnItemSelected(func(context *guigui.Context, menuIndex, itemIndex int) {
		if menuIndex < 0 || menuIndex >= len(popupItems) {
			return
		}
		ms := popupItems[menuIndex]
		if itemIndex < 0 || itemIndex >= len(ms) {
			return
		}
		value := ms[itemIndex].Value
		if menuIndex >= builtinMenuCount {
			guigui.DispatchEvent(m, menubarEventExtraItemSelected, value)
			return
		}
		var key guigui.EventKey
		switch value {
		case "new":
			key = menubarEventNew
		case "open":
			key = menubarEventOpen
		case "save":
			key = menubarEventSave
		case "saveas":
			key = menubarEventSaveAs
		case "undo":
			key = menubarEventUndo
		case "redo":
			key = menubarEventRedo
		case "cut":
			key = menubarEventCut
		case "copy":
			key = menubarEventCopy
		case "paste":
			key = menubarEventPaste
		case "find":
			key = menubarEventFind
		case "selectall":
			key = menubarEventSelectAll
		default:
			return
		}
		guigui.DispatchEvent(m, key)
	})
	return nil
}

func (m *Menubar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&m.menubar, widgetBounds.Bounds())
}

func (m *Menubar) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return m.menubar.Measure(context, constraints)
}
