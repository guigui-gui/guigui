// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	_ "github.com/guigui-gui/guigui/basicwidget/cjkfont"
)

type Root struct {
	guigui.DefaultWidget

	background    basicwidget.Background
	menubar       editorMenubar
	editor        basicwidget.TextInput
	statusBar     statusBar
	findDialog    findDialog
	confirmDialog confirmDialog
	infoDialog    infoDialog

	doc           Document
	initialText   string
	wordWrap      bool
	inited        bool
	exitRequested bool
	exitAfterSave bool
	openAfterSave bool
	newAfterSave  bool

	confirmKind confirmKind

	pendingOpen <-chan fileResult
	pendingSave <-chan fileResult

	layoutItems []guigui.LinearLayoutItem
}

// confirmKind identifies which action triggered the open confirm dialog.
// The handler set by [Root.Build] uses it to dispatch the user's choice.
type confirmKind int

const (
	confirmKindNone confirmKind = iota
	confirmKindExit
	confirmKindNew
	confirmKindOpen
)

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.menubar)
	adder.AddWidget(&r.editor)
	adder.AddWidget(&r.statusBar)
	adder.AddWidget(&r.findDialog)
	adder.AddWidget(&r.confirmDialog)
	adder.AddWidget(&r.infoDialog)

	r.editor.SetMultiline(true)
	r.editor.SetSelectionVisibleWhenUnfocused(true)
	r.editor.SetFocusBorderVisible(false)
	r.editor.SetAutoWrap(r.wordWrap)

	if !r.inited {
		r.editor.ForceSetValue(r.initialText)
		r.initialText = ""
		r.inited = true
	}

	r.editor.OnValueChanged(func(context *guigui.Context, text string, committed bool) {
		r.doc.MarkDirty()
	})
	r.editor.OnHandleButtonInput(r.handleHotkeys)

	r.findDialog.OnFindNext(func(context *guigui.Context, query string) {
		r.findNext(query)
	})
	r.findDialog.OnFindPrev(func(context *guigui.Context, query string) {
		r.findPrev(query)
	})
	r.findDialog.OnQueryChanged(func(context *guigui.Context, query string) {
		r.updateFindCount()
	})
	r.findDialog.OnClose(func(context *guigui.Context) {
		// Hand focus back to the editor so Cmd+F (and other editor hotkeys)
		// continue to work after the popup closes.
		context.SetFocused(&r.editor, true)
	})

	r.confirmDialog.OnClose(func(context *guigui.Context, result confirmResult) {
		kind := r.confirmKind
		r.confirmKind = confirmKindNone
		if result == confirmResultCancel {
			return
		}
		save := result == confirmResultSave
		switch kind {
		case confirmKindExit:
			r.handleConfirmExit(save)
		case confirmKindNew:
			r.handleConfirmNew(save)
		case confirmKindOpen:
			r.handleConfirmOpen(save)
		}
	})

	r.menubar.SetCanSave(r.doc.Path() != "")
	r.menubar.SetCanUndo(r.editor.CanUndo())
	r.menubar.SetCanRedo(r.editor.CanRedo())
	r.menubar.SetCanCut(r.editor.CanCut())
	r.menubar.SetCanCopy(r.editor.CanCopy())
	r.menubar.SetCanPaste(r.editor.CanPaste())
	r.menubar.SetWordWrap(r.wordWrap)

	r.menubar.OnNew(func(context *guigui.Context) {
		r.actionNew()
	})
	r.menubar.OnOpen(func(context *guigui.Context) {
		r.actionOpen()
	})
	r.menubar.OnSave(func(context *guigui.Context) {
		r.actionSave()
	})
	r.menubar.OnSaveAs(func(context *guigui.Context) {
		r.actionSaveAs()
	})
	r.menubar.OnUndo(func(context *guigui.Context) {
		r.editor.Undo()
	})
	r.menubar.OnRedo(func(context *guigui.Context) {
		r.editor.Redo()
	})
	r.menubar.OnCut(func(context *guigui.Context) {
		r.editor.Cut()
	})
	r.menubar.OnCopy(func(context *guigui.Context) {
		r.editor.Copy()
	})
	r.menubar.OnPaste(func(context *guigui.Context) {
		r.editor.Paste()
	})
	r.menubar.OnFind(func(context *guigui.Context) {
		r.findDialog.SetOpen(true)
	})
	r.menubar.OnSelectAll(func(context *guigui.Context) {
		r.editor.SelectAll()
	})
	r.menubar.OnToggleWordWrap(func(context *guigui.Context) {
		r.wordWrap = !r.wordWrap
	})
	r.menubar.OnAbout(func(context *guigui.Context) {
		r.infoDialog.Open()
	})

	start, _ := r.editor.Selection()
	r.statusBar.SetText(formatPosition(r.editor.Value(), start))

	if r.findDialog.IsOpen() {
		r.updateFindCount()
	}

	context.SetWindowTitle(r.windowTitle())
	return nil
}

func (r *Root) windowTitle() string {
	name := r.doc.DisplayName()
	if r.doc.IsDirty() {
		return "*" + name + " — Text Editor"
	}
	return name + " — Text Editor"
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	b := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, b)

	u := basicwidget.UnitSize(context)
	mh := r.menubar.Measure(context, guigui.Constraints{}).Y
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.menubar,
			Size:   guigui.FixedSize(mh),
		},
		guigui.LinearLayoutItem{
			Widget: &r.editor,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.statusBar,
			Size:   guigui.FixedSize(u),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
	}).LayoutWidgets(context, b, layouter)
}

func (r *Root) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	// Drain async dialog results in Tick rather than Build so a result that
	// arrives on a tick with no rebuild request is still processed promptly;
	// Build only runs when something invalidates the widget tree, but Tick
	// runs every tick.
	if err := r.drainDialogs(); err != nil {
		slog.Error("drainDialogs", "err", err)
	}

	if r.exitRequested {
		return ebiten.Termination
	}

	// Only intercept window close when there's unsaved work. Calling
	// SetWindowClosingHandled affects the window appearance on some platforms
	// (e.g. macOS shows the edited-document indicator), so leave it off when
	// the document is clean.
	needHandled := r.doc.IsDirty()
	ebiten.SetWindowClosingHandled(needHandled)

	if ebiten.IsWindowBeingClosed() {
		if !needHandled {
			return ebiten.Termination
		}
		if !r.confirmDialog.IsOpen() {
			r.confirmKind = confirmKindExit
			r.confirmDialog.SetMessage("You have unsaved changes.")
			r.confirmDialog.SetOpen(true)
		}
	}
	return nil
}

func (r *Root) drainDialogs() error {
	var err error
	if r.pendingOpen != nil {
		select {
		case res := <-r.pendingOpen:
			r.pendingOpen = nil
			switch {
			case res.cancelled:
			case res.err != nil:
				err = errors.Join(err, fmt.Errorf("open: %w", res.err))
			default:
				text, e := r.doc.Load(res.path)
				if e != nil {
					err = errors.Join(err, fmt.Errorf("open: %w", e))
				} else {
					r.editor.ForceSetValue(text)
					// ForceSetValue triggers OnValueChanged, which marks the
					// document dirty. The freshly loaded buffer matches disk,
					// so override that.
					r.doc.MarkClean()
				}
			}
		default:
		}
	}
	if r.pendingSave != nil {
		select {
		case res := <-r.pendingSave:
			r.pendingSave = nil
			saved := false
			switch {
			case res.cancelled:
			case res.err != nil:
				err = errors.Join(err, fmt.Errorf("save: %w", res.err))
			default:
				if e := r.doc.SaveAs(res.path, r.editor.Value()); e != nil {
					err = errors.Join(err, fmt.Errorf("save: %w", e))
				} else {
					saved = true
				}
			}
			if r.exitAfterSave {
				r.exitAfterSave = false
				if saved {
					r.exitRequested = true
				}
			}
			if r.openAfterSave {
				r.openAfterSave = false
				if saved {
					r.doOpen()
				}
			}
			if r.newAfterSave {
				r.newAfterSave = false
				if saved {
					r.doNew()
				}
			}
		default:
		}
	}
	return err
}

func (r *Root) actionNew() {
	if r.doc.IsDirty() {
		r.confirmKind = confirmKindNew
		r.confirmDialog.SetMessage("You have unsaved changes.")
		r.confirmDialog.SetOpen(true)
		return
	}
	r.doNew()
}

func (r *Root) handleConfirmNew(save bool) {
	if !save {
		r.doNew()
		return
	}
	r.newAfterSave = true
	r.actionSave()
	if !r.doc.IsDirty() {
		r.newAfterSave = false
		r.doNew()
	}
}

func (r *Root) doNew() {
	r.editor.ForceSetValue("")
	// ForceSetValue may have triggered OnValueChanged → MarkDirty.
	// New() resets dirty afterward.
	r.doc.New()
}

func (r *Root) actionOpen() {
	if r.doc.IsDirty() {
		r.confirmKind = confirmKindOpen
		r.confirmDialog.SetMessage("You have unsaved changes.")
		r.confirmDialog.SetOpen(true)
		return
	}
	r.doOpen()
}

func (r *Root) handleConfirmOpen(save bool) {
	if !save {
		r.doOpen()
		return
	}
	// For an untitled doc actionSave triggers an async Save As, so chain
	// the open on the save's completion (see drainDialogs).
	r.openAfterSave = true
	r.actionSave()
	if !r.doc.IsDirty() {
		r.openAfterSave = false
		r.doOpen()
	}
}

func (r *Root) handleConfirmExit(save bool) {
	if !save {
		r.exitRequested = true
		return
	}
	// For an untitled doc actionSave triggers an async Save As, so exit
	// only after the save settles (see drainDialogs).
	r.exitAfterSave = true
	r.actionSave()
	if !r.doc.IsDirty() {
		r.exitRequested = true
		r.exitAfterSave = false
	}
}

func (r *Root) doOpen() {
	if r.pendingOpen == nil {
		r.pendingOpen = openFileAsync()
	}
}

func (r *Root) actionSave() {
	if r.doc.Path() == "" {
		r.actionSaveAs()
		return
	}
	if err := r.doc.Save(r.editor.Value()); err != nil {
		slog.Error("save", "err", err)
	}
}

func (r *Root) actionSaveAs() {
	if r.pendingSave == nil {
		r.pendingSave = saveFileAsync(r.doc.DisplayName())
	}
}

func (r *Root) handleHotkeys(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !cmdPressed() {
		return guigui.HandleInputResult{}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyN):
		r.actionNew()
	case inpututil.IsKeyJustPressed(ebiten.KeyO):
		r.actionOpen()
	case inpututil.IsKeyJustPressed(ebiten.KeyS):
		r.actionSave()
	case inpututil.IsKeyJustPressed(ebiten.KeyF):
		// Toggle: Cmd+F can fire on the editor side even when the popup is
		// already shown (the popup doesn't auto-grab focus on Open).
		r.findDialog.SetOpen(!r.findDialog.IsOpen())
	default:
		return guigui.HandleInputResult{}
	}
	return guigui.HandleInputByWidget(&r.editor)
}

func (r *Root) findNext(query string) {
	defer r.updateFindCount()
	if query == "" {
		return
	}
	text := r.editor.Value()
	_, end := r.editor.Selection()
	if i := strings.Index(text[end:], query); i >= 0 {
		start := end + i
		r.editor.SetSelection(start, start+len(query))
		return
	}
	if i := strings.Index(text, query); i >= 0 {
		r.editor.SetSelection(i, i+len(query))
	}
}

func (r *Root) findPrev(query string) {
	defer r.updateFindCount()
	if query == "" {
		return
	}
	text := r.editor.Value()
	start, _ := r.editor.Selection()
	if i := strings.LastIndex(text[:start], query); i >= 0 {
		r.editor.SetSelection(i, i+len(query))
		return
	}
	if i := strings.LastIndex(text, query); i >= 0 {
		r.editor.SetSelection(i, i+len(query))
	}
}

// updateFindCount recomputes the "n of total" display from the dialog's
// current query and the editor's current selection.
func (r *Root) updateFindCount() {
	query := r.findDialog.Query()
	if query == "" {
		r.findDialog.SetCount(0, 0)
		return
	}
	text := r.editor.Value()
	matches := findAllNonOverlapping(text, query)
	if len(matches) == 0 {
		r.findDialog.SetCount(0, 0)
		return
	}
	selStart, _ := r.editor.Selection()
	cur := 0
	for i, m := range matches {
		if m == selStart {
			cur = i + 1
			break
		}
	}
	r.findDialog.SetCount(cur, len(matches))
}

func findAllNonOverlapping(text, query string) []int {
	if query == "" {
		return nil
	}
	var out []int
	var i int
	for {
		idx := strings.Index(text[i:], query)
		if idx < 0 {
			break
		}
		out = append(out, i+idx)
		i = i + idx + len(query)
	}
	return out
}

func cmdPressed() bool {
	if runtime.GOOS == "darwin" {
		return ebiten.IsKeyPressed(ebiten.KeyMeta)
	}
	return ebiten.IsKeyPressed(ebiten.KeyControl)
}

func hotkey(key string) string {
	if runtime.GOOS == "darwin" {
		return "⌘" + key
	}
	return "Ctrl+" + key
}

func hotkeyShift(key string) string {
	if runtime.GOOS == "darwin" {
		return "⇧⌘" + key
	}
	return "Ctrl+Shift+" + key
}

func main() {
	var root Root
	if len(os.Args) > 1 {
		text, err := root.doc.Load(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "load:", err)
			os.Exit(1)
		}
		root.initialText = text
	}
	op := &guigui.RunOptions{
		Title:         "Text Editor",
		WindowMinSize: image.Pt(480, 320),
	}
	if err := guigui.Run(&root, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
