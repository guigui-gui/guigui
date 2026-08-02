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
	"github.com/hajimehoshi/iro"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	_ "github.com/guigui-gui/guigui/basicwidget/cjkfont"
	"github.com/guigui-gui/guigui/example/codeeditor/internal/highlight"
	"github.com/guigui-gui/guigui/example/internal/texteditor"
)

type Root struct {
	guigui.DefaultWidget

	background    basicwidget.Background
	menubar       editorMenubar
	editor        texteditor.Editor
	statusBar     texteditor.StatusBar
	findDialog    texteditor.FindDialog
	confirmDialog texteditor.ConfirmDialog

	model Model
	doc   texteditor.Document

	codeFontFamily *basicwidget.FontFamily
	initialPath    string

	// lastCode and lastColorMode identify the inputs the cached styles
	// were derived from; lastColorMode's zero value ColorModeUnknown never
	// matches a resolved mode, so the first build always derives.
	lastCode      string
	lastColorMode ebiten.ColorMode

	highlightRanges []highlight.Range
	styles          basicwidget.TextStyles

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

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.menubar)
	adder.AddWidget(&r.editor)
	adder.AddWidget(&r.statusBar)
	adder.AddWidget(&r.findDialog)
	adder.AddWidget(&r.confirmDialog)

	r.editor.SetMultiline(true)
	r.editor.SetWrapMode(basicwidget.WrapModeNone)
	r.editor.SetFontFamily(r.codeFontFamily)
	r.editor.SetSelectionVisibleWhenUnfocused(true)
	r.editor.SetFocusBorderVisible(false)

	r.editor.OnValueChanged(func(context *guigui.Context, code string, committed bool) {
		r.model.SetCode(code)
		// Every dispatch is a real buffer change, so mark dirty regardless
		// of the committed flag; typing is uncommitted but does change the
		// buffer. Loads clear dirty afterwards (see Document.LoadInto).
		r.doc.MarkDirty()
	})
	r.editor.OnHandleButtonInput(r.handleHotkeys)

	if !r.inited {
		if r.initialPath != "" {
			if err := r.doc.LoadInto(r.initialPath, &r.editor); err != nil {
				slog.Error("load", "err", err)
			} else {
				// On the first build the value-changed handler may not have
				// been dispatched for the streamed value yet, so sync the
				// model explicitly; the SetValue below must not overwrite
				// the loaded value with the model's sample code.
				r.model.SetCode(r.editor.Value())
			}
			r.initialPath = ""
		} else {
			// Seed the sample code synchronously. The value-changed dispatch
			// marks the document dirty, so reset it afterwards: the
			// untouched sample must not trigger the unsaved-changes flow.
			r.editor.ForceSetValue(r.model.Code())
			r.doc.New()
		}
		context.SetFocused(&r.editor, true)
		r.inited = true
	}

	// The model owns the code only; the ranged styles are derived from it.
	// SetValue comes first, then the derived styles replace whatever ranged
	// styles the widget accumulated.
	r.editor.SetValue(r.model.Code())
	r.applyHighlight(context)

	r.menubar.SetCanSave(r.doc.Path() != "")
	r.menubar.SetCanUndo(r.editor.CanUndo())
	r.menubar.SetCanRedo(r.editor.CanRedo())
	r.menubar.SetCanCut(r.editor.CanCut())
	r.menubar.SetCanCopy(r.editor.CanCopy())
	r.menubar.SetCanPaste(r.editor.CanPaste())

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
	r.menubar.OnSelectAll(func(context *guigui.Context) {
		r.editor.SelectAll()
	})
	r.menubar.OnFind(func(context *guigui.Context) {
		r.findDialog.SetOpen(true)
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
		// Hand focus back to the editor so Cmd/Ctrl+F (and other editor
		// hotkeys) continue to work after the popup closes.
		context.SetFocused(&r.editor, true)
	})

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
		return "*" + name + " — Code Editor"
	}
	return name + " — Code Editor"
}

// applyHighlight installs the ranged color overrides derived from the
// current code.
func (r *Root) applyHighlight(context *guigui.Context) {
	code := r.model.Code()
	colorMode := context.ColorMode()
	if code != r.lastCode || colorMode != r.lastColorMode {
		r.lastCode = code
		r.lastColorMode = colorMode
		r.highlightRanges = highlight.AppendRanges(r.highlightRanges[:0], code)
		r.styles = basicwidget.TextStyles{}
		for _, hr := range r.highlightRanges {
			r.styles.SetColorInRange(hr.StartInBytes, hr.EndInBytes, colorForKind(hr.Kind, colorMode))
		}
	}
	r.editor.SetOverrideStyles(&r.styles)
}

// The syntax classes form a uniform palette in Oklch: every class shares
// one lightness and one chroma per color mode, and the hues are evenly
// spaced around the hue circle.
const (
	syntaxHueBaseInDegrees = 20
	syntaxHueStepInDegrees = 72

	syntaxLightnessInLightMode = 0.6
	syntaxChromaInLightMode    = 0.12
	syntaxLightnessInDarkMode  = 0.75
	syntaxChromaInDarkMode     = 0.14
)

// syntaxHueIndex returns the hue circle position of a syntax class.
func syntaxHueIndex(kind highlight.Kind) int {
	switch kind {
	case highlight.KindString:
		return 0 // Red.
	case highlight.KindDeclName:
		return 1 // Yellow-green.
	case highlight.KindComment:
		return 2 // Green.
	case highlight.KindNumber:
		return 3 // Blue.
	case highlight.KindKeyword:
		return 4 // Purple.
	}
	return 0
}

// colorForKind returns the text color of a syntax class for the color mode.
func colorForKind(kind highlight.Kind, colorMode ebiten.ColorMode) color.Color {
	l := syntaxLightnessInLightMode
	c := syntaxChromaInLightMode
	if colorMode == ebiten.ColorModeDark {
		l = syntaxLightnessInDarkMode
		c = syntaxChromaInDarkMode
	}
	h := float64(syntaxHueBaseInDegrees + syntaxHueIndex(kind)*syntaxHueStepInDegrees)
	return iro.ColorFromOklch(l, c, h, 1).SRGBColor()
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
	// which syncs the model and marks dirty. New() resets dirty afterward.
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
			Description: "Go source",
			Extensions:  []string{"go"},
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

func (r *Root) handleHotkeys(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	// Tab is not delivered as an input character, so insert it explicitly.
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		r.editor.ReplaceValueAtSelection("\t")
		return guigui.HandleInputByWidget(&r.editor)
	}
	if !texteditor.CmdPressed() {
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
		// Toggle: Cmd/Ctrl+F can fire on the editor side even when the popup
		// is already shown (the popup doesn't auto-grab focus on Open).
		r.findDialog.SetOpen(!r.findDialog.IsOpen())
	default:
		return guigui.HandleInputResult{}
	}
	return guigui.HandleInputByWidget(&r.editor)
}

// jetBrainsMonoTTFGz is the gzip-compressed TrueType data of the JetBrains
// Mono variable font, licensed under the SIL Open Font License, Version 1.1
// (see README.md).
//
//go:embed JetBrainsMono.ttf.gz
var jetBrainsMonoTTFGz []byte

func newCodeFontFamily() (*basicwidget.FontFamily, error) {
	r, err := gzip.NewReader(bytes.NewReader(jetBrainsMonoTTFGz))
	if err != nil {
		return nil, err
	}
	src, err := text.NewGoTextFaceSource(r)
	if err != nil {
		return nil, err
	}
	entries := []basicwidget.FaceSourceEntry{
		{FaceSource: src},
	}
	return basicwidget.NewFontFamily(entries, nil), nil
}

func main() {
	fontFamily, err := newCodeFontFamily()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	root := &Root{
		codeFontFamily: fontFamily,
	}
	if len(os.Args) > 1 {
		// Fail fast on a bad path so users get a terminal error rather
		// than the editor opening with the sample code. The actual
		// streaming load runs inside Build once the editor widget is ready.
		if _, err := os.Stat(os.Args[1]); err != nil {
			slog.Error("load", "err", err)
			os.Exit(1)
		}
		root.initialPath = os.Args[1]
	}
	op := &guigui.RunOptions{
		Title:         "Code Editor",
		WindowMinSize: image.Pt(640, 480),
	}
	if err := guigui.Run(root, op); err != nil {
		slog.Error("guigui.Run", "err", err)
		os.Exit(1)
	}
}
