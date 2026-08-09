// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"os"
	"slices"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	_ "github.com/guigui-gui/guigui/basicwidget/cjkfont"
	"github.com/guigui-gui/guigui/example/internal/texteditor"
)

type Root struct {
	guigui.DefaultWidget

	background    basicwidget.Background
	menubar       texteditor.Menubar
	toolbar       Toolbar
	editor        texteditor.Editor
	statusBar     texteditor.StatusBar
	findDialog    texteditor.FindDialog
	confirmDialog texteditor.ConfirmDialog

	model Model
	doc   texteditor.Document

	proseFontFamily *basicwidget.FontFamily
	initialPath     string

	// insertionStyle is the application-owned pending typing style. It is
	// pushed into the editor on every build and cleared when the editor
	// reports an insertion style reset.
	insertionStyle basicwidget.TextStyle

	inited        bool
	exitRequested bool
	exitAfterSave bool
	openAfterSave bool
	newAfterSave  bool

	confirmKind confirmKind

	pendingOpen <-chan texteditor.FileResult
	pendingSave <-chan texteditor.FileResult

	// scratchBuf is reused across builds for streaming the caret's line
	// prefix to the status-bar position display.
	scratchBuf bytes.Buffer

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

// propStates describes the effective boolean style properties and the
// effective scale of the toolbar actions' target: the selection, or the
// caret's pending typing style.
type propStates struct {
	bold          boolPropState
	italic        boolPropState
	underline     boolPropState
	strikethrough boolPropState
	scale         float64
	scaleUniform  bool
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.menubar)
	adder.AddWidget(&r.toolbar)
	adder.AddWidget(&r.editor)
	adder.AddWidget(&r.statusBar)
	adder.AddWidget(&r.findDialog)
	adder.AddWidget(&r.confirmDialog)

	r.editor.SetMultiline(true)
	r.editor.SetWrapMode(basicwidget.WrapModeNormal)
	var baseStyle basicwidget.TextStyle
	baseStyle.SetFontFamily(r.proseFontFamily)
	r.editor.SetBaseStyle(&baseStyle)
	r.editor.SetRichTextEditable(true)
	r.editor.SetSelectionVisibleWhenUnfocused(true)
	r.editor.SetFocusBorderVisible(false)

	r.editor.OnValueChanged(func(context *guigui.Context, value string, committed bool) {
		r.model.SetValue(value)
		// Every dispatch is a real buffer change; syncing the styles also
		// marks the document dirty. Loads and resets clear dirty afterwards.
		r.syncStylesToModel()
	})

	r.editor.SetInsertionStyle(&r.insertionStyle)
	r.editor.OnInsertionStyleReset(func(context *guigui.Context) {
		r.insertionStyle = basicwidget.TextStyle{}
	})

	if !r.inited {
		if r.initialPath != "" {
			if err := r.doc.LoadInto(r.initialPath, &r.editor); err != nil {
				slog.Error("load", "err", err)
			} else {
				// On the first build the value-changed handler may not have
				// been dispatched for the streamed value yet, so sync the
				// model explicitly; the SetValue below must not overwrite
				// the loaded value with the model's sample.
				r.model.SetValue(r.editor.Value())
				r.model.SetStyles(basicwidget.TextStyles{})
			}
			r.initialPath = ""
		} else {
			// Seed the styled sample synchronously. The value-changed
			// dispatch writes the widget's still-empty styles back to the
			// model and marks the document dirty, so re-seed the model and
			// reset the document afterwards: the untouched sample must not
			// trigger the unsaved-changes flow.
			r.editor.ForceSetValue(r.model.Value())
			r.model.Reset()
			r.doc.New()
		}
		context.SetFocused(&r.editor, true)
		r.inited = true
	}

	// The model owns the value and its ranged styles. Each build restores
	// them into the widget: SetValue comes first, as a changed value clears
	// the widget's followed styles before the model's are installed.
	r.editor.SetValue(r.model.Value())
	styles := r.model.Styles()
	r.editor.SetOverrideStyles(&styles, false)

	states := r.currentPropStates()
	r.toolbar.SetBoldLit(states.bold.UniformlyOn())
	r.toolbar.SetItalicLit(states.italic.UniformlyOn())
	r.toolbar.SetUnderlineLit(states.underline.UniformlyOn())
	r.toolbar.SetStrikethroughLit(states.strikethrough.UniformlyOn())
	r.toolbar.SetCanUndo(r.editor.CanUndo())
	r.toolbar.SetCanRedo(r.editor.CanRedo())

	r.toolbar.OnBold(func(context *guigui.Context) {
		r.toggleBold(context)
	})
	r.toolbar.OnItalic(func(context *guigui.Context) {
		r.toggleItalic(context)
	})
	r.toolbar.OnUnderline(func(context *guigui.Context) {
		r.toggleUnderline(context)
	})
	r.toolbar.OnStrikethrough(func(context *guigui.Context) {
		r.toggleStrikethrough(context)
	})
	r.toolbar.OnTextColorSelected(func(context *guigui.Context, clr color.Color, ok bool) {
		r.applyTextColor(context, clr, ok)
	})
	r.toolbar.OnHighlightSelected(func(context *guigui.Context, clr color.Color, ok bool) {
		r.applyHighlight(context, clr, ok)
	})
	r.toolbar.OnClear(func(context *guigui.Context) {
		r.clearStyles(context)
	})
	r.toolbar.OnScaleUp(func(context *guigui.Context) {
		r.stepScale(context, true)
	})
	r.toolbar.OnScaleDown(func(context *guigui.Context) {
		r.stepScale(context, false)
	})
	r.toolbar.OnUndo(func(context *guigui.Context) {
		r.editor.Undo()
		context.SetFocused(&r.editor, true)
	})
	r.toolbar.OnRedo(func(context *guigui.Context) {
		r.editor.Redo()
		context.SetFocused(&r.editor, true)
	})

	// Saving is not implemented yet: a saved file would lose its styles, as
	// a rich text file format is still to be decided.
	r.menubar.SetCanSave(false)
	r.menubar.SetCanSaveAs(false)
	r.menubar.SetPasteWithoutStylesVisible(true)
	r.menubar.SetCanUndo(r.editor.CanUndo())
	r.menubar.SetCanRedo(r.editor.CanRedo())
	r.menubar.SetCanCut(r.editor.CanCut())
	r.menubar.SetCanCopy(r.editor.CanCopy())
	r.menubar.SetCanPaste(r.editor.CanPaste())
	r.menubar.SetExtraMenus([]texteditor.ExtraMenu{
		{
			Text: "Format",
			Items: []basicwidget.PopupMenuItem[string]{
				{Text: "Bold", Value: "bold", KeyText: texteditor.Hotkey("B")},
				{Text: "Italic", Value: "italic", KeyText: texteditor.Hotkey("I")},
				{Text: "Underline", Value: "underline", KeyText: texteditor.Hotkey("U")},
				{Text: "Strikethrough", Value: "strikethrough"},
				{Border: true},
				{Text: "Clear Styles", Value: "clearstyles"},
			},
		},
	})

	r.menubar.OnNew(func(context *guigui.Context) {
		r.actionNew()
	})
	r.menubar.OnOpen(func(context *guigui.Context) {
		r.actionOpen()
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
	r.menubar.OnPasteWithoutStyles(func(context *guigui.Context) {
		r.editor.PasteWithoutStyles()
	})
	r.menubar.OnSelectAll(func(context *guigui.Context) {
		r.editor.SelectAll()
	})
	r.menubar.OnFind(func(context *guigui.Context) {
		r.findDialog.SetOpen(true)
	})
	r.menubar.OnExtraItemSelected(func(context *guigui.Context, value string) {
		switch value {
		case "bold":
			r.toggleBold(context)
		case "italic":
			r.toggleItalic(context)
		case "underline":
			r.toggleUnderline(context)
		case "strikethrough":
			r.toggleStrikethrough(context)
		case "clearstyles":
			r.clearStyles(context)
		}
	})

	r.findDialog.OnFindNext(func(context *guigui.Context, query string) {
		r.findDialog.FindNext(&r.editor, query)
	})
	r.findDialog.OnFindPrev(func(context *guigui.Context, query string) {
		r.findDialog.FindPrev(&r.editor, query)
	})
	r.findDialog.OnQueryChanged(func(context *guigui.Context, query string) {
		r.findDialog.UpdateCount(&r.editor)
	})
	r.findDialog.OnClose(func(context *guigui.Context) {
		// Hand focus back to the editor so that typing resumes editing the
		// document after the popup closes.
		context.SetFocused(&r.editor, true)
	})

	// Saving is not implemented yet (see the File menu), so the confirm
	// dialog offers only discarding or cancelling.
	r.confirmDialog.SetSaveEnabled(false)
	r.confirmDialog.OnClose(func(context *guigui.Context, result texteditor.ConfirmResult) {
		kind := r.confirmKind
		r.confirmKind = confirmKindNone
		if result == texteditor.ConfirmResultCancel {
			return
		}
		save := result == texteditor.ConfirmResultSave
		switch kind {
		case confirmKindExit:
			r.handleConfirmExit(save)
		case confirmKindNew:
			r.handleConfirmNew(save)
		case confirmKindOpen:
			r.handleConfirmOpen(save)
		}
	})

	start, _ := r.editor.Selection()
	line := r.editor.LineIndexFromTextIndexInBytes(start)
	lineStart := r.editor.LineStartInBytes(line)
	r.scratchBuf.Reset()
	if _, err := r.editor.WriteValueRangeTo(&r.scratchBuf, lineStart, start); err != nil {
		return err
	}
	r.statusBar.SetPosition(line+1, utf8.RuneCount(r.scratchBuf.Bytes())+1)

	if r.findDialog.IsOpen() {
		r.findDialog.UpdateCount(&r.editor)
	}

	context.SetWindowTitle(r.windowTitle())
	return nil
}

func (r *Root) WriteStateKey(context *guigui.Context, w *guigui.StateKeyWriter) {
	w.WriteBool(r.doc.IsDirty())
	w.WriteString(r.doc.Path())
}

func (r *Root) windowTitle() string {
	name := r.doc.DisplayName()
	if r.doc.IsDirty() {
		return "*" + name + " — Rich Text Editor"
	}
	return name + " — Rich Text Editor"
}

// selectionRange returns the editor's selection with start not after end.
func (r *Root) selectionRange() (start, end int) {
	start, end = r.editor.Selection()
	if start > end {
		start, end = end, start
	}
	return start, end
}

// currentPropStates returns the effective style states the toolbar reflects
// and the toggle actions decide from.
func (r *Root) currentPropStates() propStates {
	start, end := r.selectionRange()
	if start != end {
		var styles basicwidget.TextStyles
		r.editor.ReadEffectiveStylesInRange(&styles, start, end)
		n := end - start
		weight, weightUniform := styles.WeightInRange(0, n, text.WeightNormal)
		italic, italicUniform := styles.ItalicInRange(0, n, false)
		underline, underlineUniform := styles.UnderlineInRange(0, n, false)
		strikethrough, strikethroughUniform := styles.StrikethroughInRange(0, n, false)
		scale, scaleUniform := styles.ScaleInRange(0, n, 1)
		return propStates{
			bold:          boolPropState{On: isBoldWeight(weight), Uniform: weightUniform},
			italic:        boolPropState{On: italic, Uniform: italicUniform},
			underline:     boolPropState{On: underline, Uniform: underlineUniform},
			strikethrough: boolPropState{On: strikethrough, Uniform: strikethroughUniform},
			scale:         scale,
			scaleUniform:  scaleUniform,
		}
	}

	var style basicwidget.TextStyle
	r.editor.ReadEffectiveStyleAt(&style, start)
	style = mergeCaretStyle(style, r.insertionStyle)
	weight, _ := style.Weight()
	italic, _ := style.Italic()
	underline, _ := style.Underline()
	strikethrough, _ := style.Strikethrough()
	scale, ok := style.Scale()
	if !ok {
		scale = 1
	}
	return propStates{
		bold:          boolPropState{On: isBoldWeight(weight), Uniform: true},
		italic:        boolPropState{On: italic, Uniform: true},
		underline:     boolPropState{On: underline, Uniform: true},
		strikethrough: boolPropState{On: strikethrough, Uniform: true},
		scale:         scale,
		scaleUniform:  true,
	}
}

// mergeCaretStyle returns effective with pending's set properties laid over,
// so a pending property wins where set.
func mergeCaretStyle(effective, pending basicwidget.TextStyle) basicwidget.TextStyle {
	merged := effective
	if weight, ok := pending.Weight(); ok {
		merged.SetWeight(weight)
	}
	if italic, ok := pending.Italic(); ok {
		merged.SetItalic(italic)
	}
	if underline, ok := pending.Underline(); ok {
		merged.SetUnderline(underline)
	}
	if strikethrough, ok := pending.Strikethrough(); ok {
		merged.SetStrikethrough(strikethrough)
	}
	if scale, ok := pending.Scale(); ok {
		merged.SetScale(scale)
	}
	if clr, ok := pending.Color(); ok {
		merged.SetColor(clr)
	}
	if clr, ok := pending.BackgroundColor(); ok {
		merged.SetBackgroundColor(clr)
	}
	return merged
}

// syncStylesToModel writes the widget's current ranged style overrides back
// to the model, so the next build's restore does not revert them, and marks
// the document dirty.
func (r *Root) syncStylesToModel() {
	var styles basicwidget.TextStyles
	r.editor.ReadOverrideStyles(&styles)
	r.model.SetStyles(styles)
	r.doc.MarkDirty()
}

// applyToggle applies a boolean toggle action: over the selection as ranged
// overrides, or at a collapsed caret as the pending typing style. The
// direction comes from the property's effective state: uniformly on turns
// off with an explicit off-override, anything else turns on.
func (r *Root) applyToggle(context *guigui.Context, state boolPropState, applyToRange func(styles *basicwidget.TextStyles, n int, on bool), applyToCaret func(style *basicwidget.TextStyle, on bool)) {
	on := state.WillToggleOn()
	start, end := r.selectionRange()
	if start != end {
		var styles basicwidget.TextStyles
		r.editor.ReadOverrideStylesInRange(&styles, start, end)
		applyToRange(&styles, end-start, on)
		r.editor.SetOverrideStylesInRange(&styles, start, end, true)
		r.syncStylesToModel()
	} else {
		applyToCaret(&r.insertionStyle, on)
	}
	context.SetFocused(&r.editor, true)
}

func (r *Root) toggleBold(context *guigui.Context) {
	r.applyToggle(context, r.currentPropStates().bold,
		func(styles *basicwidget.TextStyles, n int, on bool) {
			weight := text.WeightNormal
			if on {
				weight = text.WeightBold
			}
			styles.SetWeightInRange(0, n, weight)
		},
		func(style *basicwidget.TextStyle, on bool) {
			weight := text.WeightNormal
			if on {
				weight = text.WeightBold
			}
			style.SetWeight(weight)
		})
}

func (r *Root) toggleItalic(context *guigui.Context) {
	r.applyToggle(context, r.currentPropStates().italic,
		func(styles *basicwidget.TextStyles, n int, on bool) {
			styles.SetItalicInRange(0, n, on)
		},
		func(style *basicwidget.TextStyle, on bool) {
			style.SetItalic(on)
		})
}

func (r *Root) toggleUnderline(context *guigui.Context) {
	r.applyToggle(context, r.currentPropStates().underline,
		func(styles *basicwidget.TextStyles, n int, on bool) {
			styles.SetUnderlineInRange(0, n, on)
		},
		func(style *basicwidget.TextStyle, on bool) {
			style.SetUnderline(on)
		})
}

func (r *Root) toggleStrikethrough(context *guigui.Context) {
	r.applyToggle(context, r.currentPropStates().strikethrough,
		func(styles *basicwidget.TextStyles, n int, on bool) {
			styles.SetStrikethroughInRange(0, n, on)
		},
		func(style *basicwidget.TextStyle, on bool) {
			style.SetStrikethrough(on)
		})
}

// applyTextColor applies a text color popup selection. ok is false for the
// default entry, which clears the property.
func (r *Root) applyTextColor(context *guigui.Context, clr color.Color, ok bool) {
	start, end := r.selectionRange()
	if start != end {
		var styles basicwidget.TextStyles
		r.editor.ReadOverrideStylesInRange(&styles, start, end)
		if ok {
			styles.SetColorInRange(0, end-start, clr)
		} else {
			styles.UnsetColorInRange(0, end-start)
		}
		r.editor.SetOverrideStylesInRange(&styles, start, end, true)
		r.syncStylesToModel()
	} else {
		if ok {
			r.insertionStyle.SetColor(clr)
		} else {
			r.insertionStyle.UnsetColor()
		}
	}
	context.SetFocused(&r.editor, true)
}

// applyHighlight applies a highlight popup selection. ok is false for the
// default entry, which clears the property.
func (r *Root) applyHighlight(context *guigui.Context, clr color.Color, ok bool) {
	start, end := r.selectionRange()
	if start != end {
		var styles basicwidget.TextStyles
		r.editor.ReadOverrideStylesInRange(&styles, start, end)
		if ok {
			styles.SetBackgroundColorInRange(0, end-start, clr)
		} else {
			styles.UnsetBackgroundColorInRange(0, end-start)
		}
		r.editor.SetOverrideStylesInRange(&styles, start, end, true)
		r.syncStylesToModel()
	} else {
		if ok {
			r.insertionStyle.SetBackgroundColor(clr)
		} else {
			r.insertionStyle.UnsetBackgroundColor()
		}
	}
	context.SetFocused(&r.editor, true)
}

// clearStyles resets the style overrides over the selection, or the pending
// typing style at a collapsed caret.
func (r *Root) clearStyles(context *guigui.Context) {
	start, end := r.selectionRange()
	if start != end {
		var styles basicwidget.TextStyles
		r.editor.SetOverrideStylesInRange(&styles, start, end, true)
		r.syncStylesToModel()
	} else {
		r.insertionStyle = basicwidget.TextStyle{}
	}
	context.SetFocused(&r.editor, true)
}

// stepScale steps the scale override through the ladder, up or down.
func (r *Root) stepScale(context *guigui.Context, up bool) {
	states := r.currentPropStates()
	var scale float64
	if up {
		scale = scaleUp(states.scale, states.scaleUniform)
	} else {
		scale = scaleDown(states.scale, states.scaleUniform)
	}
	start, end := r.selectionRange()
	if start != end {
		var styles basicwidget.TextStyles
		r.editor.ReadOverrideStylesInRange(&styles, start, end)
		styles.SetScaleInRange(0, end-start, scale)
		r.editor.SetOverrideStylesInRange(&styles, start, end, true)
		r.syncStylesToModel()
	} else {
		r.insertionStyle.SetScale(scale)
	}
	context.SetFocused(&r.editor, true)
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
			case res.Cancelled:
			case res.Err != nil:
				err = errors.Join(err, fmt.Errorf("open: %w", res.Err))
			default:
				// ReadValueFrom dispatches the value-changed handler, which
				// syncs the model and marks dirty; LoadInto re-clears dirty
				// after streaming.
				if e := r.doc.LoadInto(res.Path, &r.editor); e != nil {
					err = errors.Join(err, fmt.Errorf("open: %w", e))
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
			case res.Cancelled:
			case res.Err != nil:
				err = errors.Join(err, fmt.Errorf("save: %w", res.Err))
			default:
				if e := r.doc.SaveAs(res.Path, &r.editor); e != nil {
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
	// ForceSetValue dispatches the value-changed handler synchronously,
	// which syncs the model's value and now-empty styles and marks dirty.
	// New() resets dirty afterward.
	r.editor.ForceSetValue("")
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
		r.pendingOpen = texteditor.OpenFileAsync(&texteditor.FileFilter{
			Description: "Text",
			Extensions:  []string{"txt"},
		})
	}
}

func (r *Root) actionSave() {
	if r.doc.Path() == "" {
		r.actionSaveAs()
		return
	}
	if err := r.doc.Save(&r.editor); err != nil {
		slog.Error("save", "err", err)
	}
}

func (r *Root) actionSaveAs() {
	if r.pendingSave == nil {
		r.pendingSave = texteditor.SaveFileAsync(r.doc.DisplayName())
	}
}

// HandleButtonInput handles the application-wide shortcuts. They live on the
// root rather than on the editor so that they keep working regardless of
// which widget within the focused path receives the raw key input first.
func (r *Root) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !texteditor.CmdPressed() {
		return guigui.HandleInputResult{}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyN):
		r.actionNew()
	case inpututil.IsKeyJustPressed(ebiten.KeyO):
		r.actionOpen()
	case inpututil.IsKeyJustPressed(ebiten.KeyF):
		// The find dialog closes itself on Cmd/Ctrl+F, and its query input
		// gets the key first while it is open, so this only ever opens.
		r.findDialog.SetOpen(true)
	case inpututil.IsKeyJustPressed(ebiten.KeyB):
		r.toggleBold(context)
	case inpututil.IsKeyJustPressed(ebiten.KeyI):
		r.toggleItalic(context)
	case inpututil.IsKeyJustPressed(ebiten.KeyU):
		r.toggleUnderline(context)
	default:
		return guigui.HandleInputResult{}
	}
	return guigui.HandleInputByWidget(r)
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	b := widgetBounds.Bounds()
	layouter.LayoutWidget(&r.background, b)

	u := basicwidget.UnitSize(context)
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.menubar,
		},
		guigui.LinearLayoutItem{
			Widget: &r.toolbar,
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

// interVariableItalicTTFGz is the gzip-compressed TrueType data of the Inter
// Variable Italic font, licensed under the SIL Open Font License, Version
// 1.1 (see README.md). It complements the bundled upright Inter face so
// that the italic style renders with a true italic face.
//
//go:embed InterVariable-Italic.ttf.gz
var interVariableItalicTTFGz []byte

func newProseFontFamily() (*basicwidget.FontFamily, error) {
	reader, err := gzip.NewReader(bytes.NewReader(interVariableItalicTTFGz))
	if err != nil {
		return nil, err
	}
	italicSrc, err := text.NewGoTextFaceSource(reader)
	if err != nil {
		return nil, err
	}
	entries := []basicwidget.FaceSourceEntry{
		basicwidget.DefaultFaceSourceEntry(),
		{FaceSource: italicSrc},
	}
	return basicwidget.NewFontFamily(entries, nil), nil
}

func main() {
	fontFamily, err := newProseFontFamily()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	root := &Root{
		proseFontFamily: fontFamily,
	}
	if len(os.Args) > 1 {
		// Fail fast on a bad path so users get a terminal error rather than
		// the editor opening with the sample. The actual streaming load runs
		// inside Build once the editor widget is ready.
		if _, err := os.Stat(os.Args[1]); err != nil {
			slog.Error("load", "err", err)
			os.Exit(1)
		}
		root.initialPath = os.Args[1]
	}
	op := &guigui.RunOptions{
		Title:         "Rich Text Editor",
		WindowMinSize: image.Pt(640, 480),
	}
	if err := guigui.Run(root, op); err != nil {
		slog.Error("guigui.Run", "err", err)
		os.Exit(1)
	}
}
