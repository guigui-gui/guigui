// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

var (
	editorMenubarEventNew        = guigui.GenerateEventKey()
	editorMenubarEventOpen       = guigui.GenerateEventKey()
	editorMenubarEventSave       = guigui.GenerateEventKey()
	editorMenubarEventSaveAs     = guigui.GenerateEventKey()
	editorMenubarEventUndo       = guigui.GenerateEventKey()
	editorMenubarEventRedo       = guigui.GenerateEventKey()
	editorMenubarEventCut        = guigui.GenerateEventKey()
	editorMenubarEventCopy       = guigui.GenerateEventKey()
	editorMenubarEventPaste      = guigui.GenerateEventKey()
	editorMenubarEventFind       = guigui.GenerateEventKey()
	editorMenubarEventSelectAll  = guigui.GenerateEventKey()
	editorMenubarEventToggleWrap = guigui.GenerateEventKey()
	editorMenubarEventAbout      = guigui.GenerateEventKey()
)

// editorMenubar is the application's menubar widget. It wraps
// [basicwidget.Menubar] with the texteditor's specific menu structure and
// exposes typed event handlers (OnNew, OnOpen, ...) so the Root widget can
// register actions without dealing with menu/item indices.
type editorMenubar struct {
	guigui.DefaultWidget

	menubar basicwidget.Menubar[string]

	canSave  bool
	canUndo  bool
	canRedo  bool
	canCut   bool
	canCopy  bool
	canPaste bool
	wordWrap bool
}

func (m *editorMenubar) SetCanSave(b bool) {
	m.canSave = b
}

func (m *editorMenubar) SetCanUndo(b bool) {
	m.canUndo = b
}

func (m *editorMenubar) SetCanRedo(b bool) {
	m.canRedo = b
}

func (m *editorMenubar) SetCanCut(b bool) {
	m.canCut = b
}

func (m *editorMenubar) SetCanCopy(b bool) {
	m.canCopy = b
}

func (m *editorMenubar) SetCanPaste(b bool) {
	m.canPaste = b
}

func (m *editorMenubar) SetWordWrap(b bool) {
	m.wordWrap = b
}

func (m *editorMenubar) OnNew(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventNew, fn)
}

func (m *editorMenubar) OnOpen(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventOpen, fn)
}

func (m *editorMenubar) OnSave(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventSave, fn)
}

func (m *editorMenubar) OnSaveAs(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventSaveAs, fn)
}

func (m *editorMenubar) OnUndo(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventUndo, fn)
}

func (m *editorMenubar) OnRedo(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventRedo, fn)
}

func (m *editorMenubar) OnCut(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventCut, fn)
}

func (m *editorMenubar) OnCopy(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventCopy, fn)
}

func (m *editorMenubar) OnPaste(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventPaste, fn)
}

func (m *editorMenubar) OnFind(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventFind, fn)
}

func (m *editorMenubar) OnSelectAll(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventSelectAll, fn)
}

func (m *editorMenubar) OnToggleWordWrap(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventToggleWrap, fn)
}

func (m *editorMenubar) OnAbout(fn func(context *guigui.Context)) {
	guigui.SetEventHandler(m, editorMenubarEventAbout, fn)
}

func (m *editorMenubar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&m.menubar)

	m.menubar.SetItems([]basicwidget.MenubarItem{
		{Text: "File"},
		{Text: "Edit"},
		{Text: "Format"},
		{Text: "Help"},
	})

	popupItems := [][]basicwidget.PopupMenuItem[string]{
		{
			{Text: "New", Value: "new", KeyText: hotkey("N")},
			{Text: "Open…", Value: "open", KeyText: hotkey("O")},
			{Border: true},
			{Text: "Save", Value: "save", KeyText: hotkey("S"), Disabled: !m.canSave},
			{Text: "Save As…", Value: "saveas"},
		},
		{
			{Text: "Undo", Value: "undo", KeyText: hotkey("Z"), Disabled: !m.canUndo},
			{Text: "Redo", Value: "redo", KeyText: hotkeyShift("Z"), Disabled: !m.canRedo},
			{Border: true},
			{Text: "Cut", Value: "cut", Disabled: !m.canCut},
			{Text: "Copy", Value: "copy", Disabled: !m.canCopy},
			{Text: "Paste", Value: "paste", Disabled: !m.canPaste},
			{Border: true},
			{Text: "Find…", Value: "find", KeyText: hotkey("F")},
			{Border: true},
			{Text: "Select All", Value: "selectall", KeyText: hotkey("A")},
		},
		{
			{Text: "Word Wrap", Value: "wrap", Checked: m.wordWrap},
		},
		{
			{Text: "About", Value: "about"},
		},
	}
	for i, items := range popupItems {
		m.menubar.PopupMenuAt(i).SetItems(items)
	}
	m.menubar.PopupMenuAt(2).SetReservesCheckmarkSpace(true)

	m.menubar.OnItemSelected(func(context *guigui.Context, menuIndex, itemIndex int) {
		if menuIndex < 0 || menuIndex >= len(popupItems) {
			return
		}
		ms := popupItems[menuIndex]
		if itemIndex < 0 || itemIndex >= len(ms) {
			return
		}
		var key guigui.EventKey
		switch ms[itemIndex].Value {
		case "new":
			key = editorMenubarEventNew
		case "open":
			key = editorMenubarEventOpen
		case "save":
			key = editorMenubarEventSave
		case "saveas":
			key = editorMenubarEventSaveAs
		case "undo":
			key = editorMenubarEventUndo
		case "redo":
			key = editorMenubarEventRedo
		case "cut":
			key = editorMenubarEventCut
		case "copy":
			key = editorMenubarEventCopy
		case "paste":
			key = editorMenubarEventPaste
		case "find":
			key = editorMenubarEventFind
		case "selectall":
			key = editorMenubarEventSelectAll
		case "wrap":
			key = editorMenubarEventToggleWrap
		case "about":
			key = editorMenubarEventAbout
		default:
			return
		}
		guigui.DispatchEvent(m, key)
	})
	return nil
}

func (m *editorMenubar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&m.menubar, widgetBounds.Bounds())
}

func (m *editorMenubar) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return m.menubar.Measure(context, constraints)
}
