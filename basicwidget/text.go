// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"log/slog"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/rivo/uniseg"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
	"github.com/guigui-gui/guigui/internal/clipboard"
)

type HorizontalAlign int

const (
	HorizontalAlignStart  HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignStart)
	HorizontalAlignCenter HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignCenter)
	HorizontalAlignEnd    HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignEnd)
	HorizontalAlignLeft   HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignLeft)
	HorizontalAlignRight  HorizontalAlign = HorizontalAlign(textutil.HorizontalAlignRight)
)

type VerticalAlign int

const (
	VerticalAlignTop    VerticalAlign = VerticalAlign(textutil.VerticalAlignTop)
	VerticalAlignMiddle VerticalAlign = VerticalAlign(textutil.VerticalAlignMiddle)
	VerticalAlignBottom VerticalAlign = VerticalAlign(textutil.VerticalAlignBottom)
)

const (
	textEventKeyJustPressed = "keyJustPressed"
	textEventValueChanged   = "valueChanged"
)

func isMouseButtonRepeating(button ebiten.MouseButton) bool {
	return repeat(inpututil.MouseButtonPressDuration(button))
}

func isKeyRepeating(key ebiten.Key) bool {
	return repeat(inpututil.KeyPressDuration(key))
}

func repeat(duration int) bool {
	if duration == 1 {
		return true
	}
	delay := ebiten.TPS() * 24 / 60
	if duration < delay {
		return false
	}
	return (duration-delay)%4 == 0
}

func findWordBoundaries(text string, idx int) (start, end int) {
	start = idx
	end = idx

	word, _, _ := uniseg.FirstWordInString(text[idx:], -1)
	end += len(word)

	for {
		word, _, _ = uniseg.FirstWordInString(text[start:], -1)
		if start+len(word) < end {
			start += len(word)
			break
		}
		if start == 0 {
			break
		}
		_, l := utf8.DecodeLastRuneInString(text[:start])
		start -= l
	}

	return start, end
}

type Text struct {
	guigui.DefaultWidget

	field         textinput.Field
	nextTextSet   bool
	nextText      string
	nextSelectAll bool

	hAlign      HorizontalAlign
	vAlign      VerticalAlign
	color       color.Color
	transparent float64
	locales     []language.Tag
	scaleMinus1 float64
	bold        bool
	tabular     bool
	tabWidth    float64

	selectable       bool
	editable         bool
	multiline        bool
	autoWrap         bool
	keepTailingSpace bool

	selectionDragStartPlus1 int
	selectionDragEndPlus1   int

	// selectionShiftIndexPlus1 is the index (+1) of the selection that is moved by Shift and arrow keys.
	selectionShiftIndexPlus1 int

	dragging bool

	clickCount         int
	lastClickTick      int64
	lastClickTextIndex int

	cursor textCursor

	tmpClipboard string

	cachedTextSizes [4][4]cachedTextSizeEntry
	lastFace        text.Face
	lastScale       float64

	tmpLocales []language.Tag

	prevStart int
	prevEnd   int
}

type cachedTextSizeEntry struct {
	// 0 indicates that the entry is invalid.
	constraintWidth int

	size image.Point
}

type textSizeCacheKey int

func newTextSizeCacheKey(autoWrap, bold bool) textSizeCacheKey {
	var key textSizeCacheKey
	if autoWrap {
		key |= 1 << 0
	}
	if bold {
		key |= 1 << 1
	}
	return key
}

func (t *Text) SetOnValueChanged(f func(context *guigui.Context, text string, committed bool)) {
	guigui.SetEventHandler(t, textEventValueChanged, f)
}

func (t *Text) SetOnKeyJustPressed(f func(context *guigui.Context, key ebiten.Key)) {
	guigui.SetEventHandler(t, textEventKeyJustPressed, f)
}

func (t *Text) resetCachedTextSize() {
	clear(t.cachedTextSizes[:])
}

func (t *Text) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if t.selectable || t.editable {
		adder.AddChild(&t.cursor)
	}

	if f := t.face(context, false); t.lastFace != f {
		t.lastFace = f
		t.resetCachedTextSize()
	}
	if t.lastScale != context.Scale() {
		t.lastScale = context.Scale()
		t.resetCachedTextSize()
	}

	context.SetPassThrough(&t.cursor, true)
	context.SetFloat(&t.cursor, true)

	if t.selectable || t.editable {
		t.cursor.text = t
	}

	return nil
}

func (t *Text) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	vb := widgetBounds.VisibleBounds()
	vb.Min.X -= textCursorWidth(context) / 2
	vb.Max.X += textCursorWidth(context) / 2
	layouter.LayoutWidget(&t.cursor, t.cursorBounds(context, widgetBounds).Intersect(vb))
}

func (t *Text) SetSelectable(selectable bool) {
	if t.selectable == selectable {
		return
	}
	t.selectable = selectable
	t.selectionDragStartPlus1 = 0
	t.selectionDragEndPlus1 = 0
	t.selectionShiftIndexPlus1 = 0
	if !t.selectable {
		t.setTextAndSelection(t.field.Text(), 0, 0, -1, false)
	}
	guigui.RequestRedraw(t)
}

func (t *Text) Value() string {
	if t.nextTextSet {
		return t.nextText
	}
	return t.field.Text()
}

func (t *Text) SetValue(text string) {
	if t.nextTextSet && t.nextText == text {
		return
	}
	if !t.nextTextSet && t.field.Text() == text {
		return
	}

	// When a user is editing, the text should not be changed.
	// Update the actual value later.
	if t.editable {
		t.nextText = text
		t.nextTextSet = true
	} else {
		t.setText(text)
	}
	t.resetCachedTextSize()
}

func (t *Text) ForceSetValue(text string) {
	t.setText(text)
}

func (t *Text) ReplaceValueAtSelection(text string) {
	if text == "" {
		return
	}
	start, end := t.field.Selection()
	newText := t.field.Text()[:start] + text + t.field.Text()[end:]
	t.setTextAndSelection(newText, start+len(text), start+len(text), -1, true)
	t.nextText = ""
	t.nextTextSet = false
	t.resetCachedTextSize()
}

func (t *Text) CommitWithCurrentInputValue() {
	changed := t.setText(t.field.Text())
	if changed {
		return
	}
	// Fire the event even if the text is not changed.
	guigui.DispatchEvent(t, textEventValueChanged, t.field.Text(), true)
}

func (t *Text) setText(text string) bool {
	start, end := t.field.Selection()
	start = min(start, len(text))
	end = min(end, len(text))
	changed := t.setTextAndSelection(text, start, end, -1, false)
	t.nextText = ""
	t.nextTextSet = false
	return changed
}

func (t *Text) selectAll() {
	if t.nextTextSet {
		t.nextSelectAll = true
		return
	}
	t.doSelectAll()
}

func (t *Text) doSelectAll() {
	t.setTextAndSelection(t.field.Text(), 0, len(t.field.Text()), -1, false)
}

func (t *Text) setTextAndSelection(text string, start, end int, shiftIndex int, adjustScroll bool) bool {
	if !t.multiline {
		text, start, end, shiftIndex = replaceNewLinesWithSpace(text, start, end, shiftIndex)
	}

	t.selectionShiftIndexPlus1 = shiftIndex + 1
	if start > end {
		start, end = end, start
	}

	textChanged := t.field.Text() != text
	if s, e := t.field.Selection(); t.field.Text() == text && s == start && e == end {
		return false
	}
	t.field.SetTextAndSelection(text, start, end)
	guigui.RequestRedraw(t)
	if textChanged {
		t.resetCachedTextSize()
		guigui.DispatchEvent(t, textEventValueChanged, t.field.Text(), false)
	}

	if !adjustScroll {
		t.prevStart = start
		t.prevEnd = end
	}

	return true
}

func (t *Text) SetLocales(locales []language.Tag) {
	if slices.Equal(t.locales, locales) {
		return
	}

	t.locales = append([]language.Tag(nil), locales...)
	guigui.RequestRedraw(t)
}

func (t *Text) SetBold(bold bool) {
	if t.bold == bold {
		return
	}

	t.bold = bold
	guigui.RequestRedraw(t)
}

func (t *Text) SetTabular(tabular bool) {
	if t.tabular == tabular {
		return
	}

	t.tabular = tabular
	guigui.RequestRedraw(t)
}

func (t *Text) SetTabWidth(tabWidth float64) {
	if t.tabWidth == tabWidth {
		return
	}
	t.tabWidth = tabWidth
	t.resetCachedTextSize()
	guigui.RequestRedraw(t)
}

func (t *Text) actualTabWidth(context *guigui.Context) float64 {
	if t.tabWidth > 0 {
		return t.tabWidth
	}
	face := t.face(context, false)
	return text.Advance("        ", face)
}

func (t *Text) SetScale(scale float64) {
	if t.scaleMinus1 == scale-1 {
		return
	}

	t.scaleMinus1 = scale - 1
	guigui.RequestRedraw(t)
}

func (t *Text) HorizontalAlign() HorizontalAlign {
	return t.hAlign
}

func (t *Text) SetHorizontalAlign(align HorizontalAlign) {
	if t.hAlign == align {
		return
	}

	t.hAlign = align
	guigui.RequestRedraw(t)
}

func (t *Text) VerticalAlign() VerticalAlign {
	return t.vAlign
}

func (t *Text) SetVerticalAlign(align VerticalAlign) {
	if t.vAlign == align {
		return
	}

	t.vAlign = align
	guigui.RequestRedraw(t)
}

func (t *Text) SetColor(color color.Color) {
	if draw.EqualColor(t.color, color) {
		return
	}

	t.color = color
	guigui.RequestRedraw(t)
}

func (t *Text) SetOpacity(opacity float64) {
	if 1-t.transparent == opacity {
		return
	}

	t.transparent = 1 - opacity
	guigui.RequestRedraw(t)
}

func (t *Text) IsEditable() bool {
	return t.editable
}

func (t *Text) SetEditable(editable bool) {
	if t.editable == editable {
		return
	}

	if editable {
		t.selectionDragStartPlus1 = 0
		t.selectionDragEndPlus1 = 0
		t.selectionShiftIndexPlus1 = 0
	}
	t.editable = editable
	guigui.RequestRedraw(t)
}

func (t *Text) IsMultiline() bool {
	return t.multiline
}

func (t *Text) SetMultiline(multiline bool) {
	if t.multiline == multiline {
		return
	}

	t.multiline = multiline
	guigui.RequestRedraw(t)
}

func (t *Text) SetAutoWrap(autoWrap bool) {
	if t.autoWrap == autoWrap {
		return
	}

	t.autoWrap = autoWrap
	guigui.RequestRedraw(t)
}

func (t *Text) setKeepTailingSpace(keep bool) {
	if t.keepTailingSpace == keep {
		return
	}

	t.keepTailingSpace = keep
	guigui.RequestRedraw(t)
}

func (t *Text) textContentBounds(context *guigui.Context, bounds image.Rectangle) image.Rectangle {
	b := bounds
	ts := t.Measure(context, guigui.FixedWidthConstraints(b.Dx()))

	switch t.vAlign {
	case VerticalAlignTop:
		b.Max.Y = b.Min.Y + ts.Y
	case VerticalAlignMiddle:
		h := b.Dy()
		b.Min.Y += (h - ts.Y) / 2
		b.Max.Y = b.Min.Y + ts.Y
	case VerticalAlignBottom:
		b.Min.Y = b.Max.Y - ts.Y
	}

	return b
}

func (t *Text) face(context *guigui.Context, forceBold bool) text.Face {
	size := FontSize(context) * (t.scaleMinus1 + 1)
	weight := text.WeightMedium
	if t.bold || forceBold {
		weight = text.WeightBold
	}

	liga := !t.selectable && !t.editable
	tnum := t.tabular

	var lang language.Tag
	if len(t.locales) > 0 {
		lang = t.locales[0]
	} else {
		t.tmpLocales = slices.Delete(t.tmpLocales, 0, len(t.tmpLocales))
		t.tmpLocales = context.AppendLocales(t.tmpLocales)
		if len(t.tmpLocales) > 0 {
			lang = t.tmpLocales[0]
		}
	}
	return fontFace(context, size, weight, liga, tnum, lang)
}

func (t *Text) lineHeight(context *guigui.Context) float64 {
	return LineHeight(context) * (t.scaleMinus1 + 1)
}

func (t *Text) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if !t.selectable && !t.editable {
		return guigui.HandleInputResult{}
	}

	cursorPosition := image.Pt(ebiten.CursorPosition())
	if t.dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), cursorPosition, false)
			start, end := idx, idx
			if t.selectionDragStartPlus1-1 >= 0 {
				start = min(start, t.selectionDragStartPlus1-1)
			}
			if t.selectionDragEndPlus1-1 >= 0 {
				end = max(idx, t.selectionDragEndPlus1-1)
			}
			if t.setTextAndSelection(t.field.Text(), start, end, -1, true) {
				return guigui.HandleInputByWidget(t)
			} else {
				return guigui.AbortHandlingInputByWidget(t)
			}
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			t.dragging = false
			t.selectionDragStartPlus1 = 0
			t.selectionDragEndPlus1 = 0
			return guigui.HandleInputByWidget(t)
		}
		return guigui.AbortHandlingInputByWidget(t)
	}

	left := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	right := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if left || right {
		if widgetBounds.IsHitAtCursor() {
			t.handleClick(context, widgetBounds.Bounds(), cursorPosition, left)
			if left {
				return guigui.HandleInputByWidget(t)
			}
			return guigui.HandleInputResult{}
		}
		context.SetFocused(t, false)
	}

	if !context.IsFocused(t) {
		if t.field.IsFocused() {
			t.field.Blur()
			guigui.RequestRedraw(t)
		}
		return guigui.HandleInputResult{}
	}
	t.field.Focus()

	if !t.editable && !t.selectable {
		return guigui.HandleInputResult{}
	}

	return guigui.HandleInputResult{}
}

func (t *Text) handleClick(context *guigui.Context, textBounds image.Rectangle, cursorPosition image.Point, leftClick bool) {
	idx := t.textIndexFromPosition(context, textBounds, cursorPosition, false)

	if leftClick {
		if ebiten.Tick()-t.lastClickTick < int64(doubleClickLimitInTicks()) && t.lastClickTextIndex == idx {
			t.clickCount++
		} else {
			t.clickCount = 1
		}
	} else {
		t.clickCount = 1
	}

	switch t.clickCount {
	case 1:
		if leftClick {
			t.dragging = true
			t.selectionDragStartPlus1 = idx + 1
			t.selectionDragEndPlus1 = idx + 1
		} else {
			t.dragging = false
			t.selectionDragStartPlus1 = 0
			t.selectionDragEndPlus1 = 0
		}
		if leftClick || !context.IsFocusedOrHasFocusedChild(t) {
			if start, end := t.field.Selection(); start != idx || end != idx {
				t.setTextAndSelection(t.field.Text(), idx, idx, -1, false)
			}
		}
	case 2:
		t.dragging = true
		text := t.field.Text()
		start, end := findWordBoundaries(text, idx)
		t.selectionDragStartPlus1 = start + 1
		t.selectionDragEndPlus1 = end + 1
		t.setTextAndSelection(text, start, end, -1, false)
	case 3:
		t.doSelectAll()
	}

	context.SetFocused(t, true)

	t.lastClickTick = ebiten.Tick()
	t.lastClickTextIndex = idx
}

func (t *Text) textToDraw(context *guigui.Context, showComposition bool) string {
	if showComposition {
		return t.field.TextForRendering()
	}
	return t.field.Text()
}

func (t *Text) selectionToDraw(context *guigui.Context) (start, end int, ok bool) {
	s, e := t.field.Selection()
	if !t.editable {
		return s, e, true
	}
	if !context.IsFocused(t) {
		return s, e, true
	}
	cs, ce, ok := t.field.CompositionSelection()
	if !ok {
		return s, e, true
	}
	// When cs == ce, the composition already started but any conversion is not done yet.
	// In this case, put the cursor at the end of the composition.
	// TODO: This behavior might be macOS specific. Investigate this.
	if cs == ce {
		return s + ce, s + ce, true
	}
	return 0, 0, false
}

func (t *Text) compositionSelectionToDraw(context *guigui.Context) (uStart, cStart, cEnd, uEnd int, ok bool) {
	if !t.editable {
		return 0, 0, 0, 0, false
	}
	if !context.IsFocused(t) {
		return 0, 0, 0, 0, false
	}
	s, _ := t.field.Selection()
	cs, ce, ok := t.field.CompositionSelection()
	if !ok {
		return 0, 0, 0, 0, false
	}
	// When cs == ce, the composition already started but any conversion is not done yet.
	// In this case, assume the entire region is the composition.
	// TODO: This behavior might be macOS specific. Investigate this.
	l := t.field.UncommittedTextLengthInBytes()
	if cs == ce {
		return s, s, s + l, s + l, true
	}
	return s, s + cs, s + ce, s + l, true
}

func (t *Text) HandleButtonInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	// Handle a key input by user-setting callback, unless IME is working.
	if t.field.UncommittedTextLengthInBytes() == 0 {
		for _, key := range inpututil.AppendJustPressedKeys(nil) {
			guigui.DispatchEvent(t, textEventKeyJustPressed, key)
		}
	}

	if !t.selectable && !t.editable {
		return guigui.HandleInputResult{}
	}

	if t.editable {
		origText := t.field.Text()
		start, _ := t.field.Selection()
		var processed bool
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), start, false); ok {
			var err error
			processed, err = t.field.HandleInputWithBounds(image.Rect(int(pos.X), int(pos.Top), int(pos.X+1), int(pos.Bottom)))
			if err != nil {
				slog.Error(err.Error())
				return guigui.AbortHandlingInputByWidget(t)
			}
		}
		if processed {
			guigui.RequestRedraw(t)
			// Reset the cache size before adjust the scroll offset in order to get the correct text size.
			t.resetCachedTextSize()
			if t.field.Text() != origText {
				guigui.DispatchEvent(t, textEventValueChanged, t.field.Text(), false)
			}
			return guigui.HandleInputByWidget(t)
		}

		// Do not accept key inputs when compositing.
		if _, _, ok := t.field.CompositionSelection(); ok {
			return guigui.HandleInputByWidget(t)
		}

		// For Windows key binds, see:
		// https://support.microsoft.com/en-us/windows/keyboard-shortcuts-in-windows-dcc61a57-8ff0-cffe-9796-cb9706c75eec#textediting

		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
			if t.multiline {
				start, end := t.field.Selection()
				text := t.field.Text()[:start] + "\n" + t.field.Text()[end:]
				t.setTextAndSelection(text, start+len("\n"), start+len("\n"), -1, true)
			} else {
				t.commit()
			}
			return guigui.HandleInputByWidget(t)
		case isKeyRepeating(ebiten.KeyBackspace) ||
			useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyH):
			start, end := t.field.Selection()
			if start != end {
				text := t.field.Text()[:start] + t.field.Text()[end:]
				t.setTextAndSelection(text, start, start, -1, true)
			} else if start > 0 {
				text, pos := textutil.BackspaceOnGraphemes(t.field.Text(), start)
				t.setTextAndSelection(text, pos, pos, -1, true)
			}
			return guigui.HandleInputByWidget(t)
		case !useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyD) ||
			useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyD):
			// Delete
			start, end := t.field.Selection()
			if start != end {
				text := t.field.Text()[:start] + t.field.Text()[end:]
				t.setTextAndSelection(text, start, start, -1, true)
			} else if useEmacsKeybind() && end < len(t.field.Text()) {
				text, pos := textutil.DeleteOnGraphemes(t.field.Text(), end)
				t.setTextAndSelection(text, pos, pos, -1, true)
			}
			return guigui.HandleInputByWidget(t)
		case isKeyRepeating(ebiten.KeyDelete):
			// Delete one cluster
			if _, end := t.field.Selection(); end < len(t.field.Text()) {
				text, pos := textutil.DeleteOnGraphemes(t.field.Text(), end)
				t.setTextAndSelection(text, pos, pos, -1, true)
			}
			return guigui.HandleInputByWidget(t)
		case !useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyX) ||
			useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyX):
			// Cut
			t.Cut()
			return guigui.HandleInputByWidget(t)
		case !useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyV) ||
			useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyV):
			// Paste
			t.Paste()
			return guigui.HandleInputByWidget(t)
		}
	}

	switch {
	case isKeyRepeating(ebiten.KeyLeft) ||
		useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyB):
		start, end := t.field.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.selectionShiftIndexPlus1-1 == end {
				pos := textutil.PrevPositionOnGraphemes(t.field.Text(), end)
				t.setTextAndSelection(t.field.Text(), start, pos, pos, true)
			} else {
				pos := textutil.PrevPositionOnGraphemes(t.field.Text(), start)
				t.setTextAndSelection(t.field.Text(), pos, end, pos, true)
			}
		} else {
			if start != end {
				t.setTextAndSelection(t.field.Text(), start, start, -1, true)
			} else if start > 0 {
				pos := textutil.PrevPositionOnGraphemes(t.field.Text(), start)
				t.setTextAndSelection(t.field.Text(), pos, pos, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyRight) ||
		useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyF):
		start, end := t.field.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.selectionShiftIndexPlus1-1 == start {
				pos := textutil.NextPositionOnGraphemes(t.field.Text(), start)
				t.setTextAndSelection(t.field.Text(), pos, end, pos, true)
			} else {
				pos := textutil.NextPositionOnGraphemes(t.field.Text(), end)
				t.setTextAndSelection(t.field.Text(), start, pos, pos, true)
			}
		} else {
			if start != end {
				t.setTextAndSelection(t.field.Text(), end, end, -1, true)
			} else if start < len(t.field.Text()) {
				pos := textutil.NextPositionOnGraphemes(t.field.Text(), start)
				t.setTextAndSelection(t.field.Text(), pos, pos, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyUp) ||
		useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyP):
		lh := t.lineHeight(context)
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveEnd bool
		start, end := t.field.Selection()
		idx := start
		if shift && t.selectionShiftIndexPlus1-1 == end {
			idx = end
			moveEnd = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			y := (pos.Top+pos.Bottom)/2 - lh
			idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			if shift {
				if moveEnd {
					t.setTextAndSelection(t.field.Text(), start, idx, idx, true)
				} else {
					t.setTextAndSelection(t.field.Text(), idx, end, idx, true)
				}
			} else {
				t.setTextAndSelection(t.field.Text(), idx, idx, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyDown) ||
		useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyN):
		lh := t.lineHeight(context)
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		var moveStart bool
		start, end := t.field.Selection()
		idx := end
		if shift && t.selectionShiftIndexPlus1-1 == start {
			idx = start
			moveStart = true
		}
		if pos, ok := t.textPosition(context, widgetBounds.Bounds(), idx, false); ok {
			y := (pos.Top+pos.Bottom)/2 + lh
			idx := t.textIndexFromPosition(context, widgetBounds.Bounds(), image.Pt(int(pos.X), int(y)), false)
			if shift {
				if moveStart {
					t.setTextAndSelection(t.field.Text(), idx, end, idx, true)
				} else {
					t.setTextAndSelection(t.field.Text(), start, idx, idx, true)
				}
			} else {
				t.setTextAndSelection(t.field.Text(), idx, idx, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyA):
		idx := 0
		start, end := t.field.Selection()
		if i := strings.LastIndex(t.field.Text()[:start], "\n"); i >= 0 {
			idx = i + 1
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setTextAndSelection(t.field.Text(), idx, end, idx, true)
		} else {
			t.setTextAndSelection(t.field.Text(), idx, idx, -1, true)
		}
		return guigui.HandleInputByWidget(t)
	case useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyE):
		idx := len(t.field.Text())
		start, end := t.field.Selection()
		if i := strings.Index(t.field.Text()[end:], "\n"); i >= 0 {
			idx = end + i
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setTextAndSelection(t.field.Text(), start, idx, idx, true)
		} else {
			t.setTextAndSelection(t.field.Text(), idx, idx, -1, true)
		}
		return guigui.HandleInputByWidget(t)
	case !useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyA) ||
		useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyA):
		t.doSelectAll()
		return guigui.HandleInputByWidget(t)
	case !useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyC) ||
		useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyC):
		// Copy
		t.Copy()
		return guigui.HandleInputByWidget(t)
	case useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyK):
		// 'Kill' the text after the cursor or the selection.
		start, end := t.field.Selection()
		if start == end {
			end = strings.Index(t.field.Text()[start:], "\n")
			if end < 0 {
				end = len(t.field.Text())
			} else if end == 0 {
				end += start + 1
			} else {
				end += start
			}
		}
		t.tmpClipboard = t.field.Text()[start:end]
		text := t.field.Text()[:start] + t.field.Text()[end:]
		t.setTextAndSelection(text, start, start, -1, true)
		return guigui.HandleInputByWidget(t)
	case useEmacsKeybind() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyY):
		// 'Yank' the killed text.
		if t.tmpClipboard != "" {
			start, _ := t.field.Selection()
			text := t.field.Text()[:start] + t.tmpClipboard + t.field.Text()[start:]
			t.setTextAndSelection(text, start+len(t.tmpClipboard), start+len(t.tmpClipboard), -1, true)
		}
		return guigui.HandleInputByWidget(t)
	}

	return guigui.HandleInputResult{}
}

func (t *Text) commit() {
	guigui.DispatchEvent(t, textEventValueChanged, t.field.Text(), true)
	t.nextText = ""
	t.nextTextSet = false
}

func (t *Text) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if !context.IsFocused(t) && t.nextTextSet {
		if t.nextSelectAll {
			t.setTextAndSelection(t.nextText, 0, len(t.nextText), -1, false)
		} else {
			t.setText(t.nextText)
		}
		t.nextTextSet = false
		t.nextText = ""
		t.nextSelectAll = false
	}
	return nil
}

func (t *Text) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	textBounds := t.textContentBounds(context, widgetBounds.Bounds())
	if !textBounds.Overlaps(widgetBounds.VisibleBounds()) {
		return
	}

	var textColor color.Color
	if t.color != nil {
		textColor = t.color
	} else {
		textColor = basicwidgetdraw.TextColor(context.ColorMode(), context.IsEnabled(t))
	}
	if t.transparent > 0 {
		textColor = draw.ScaleAlpha(textColor, 1-t.transparent)
	}
	face := t.face(context, false)
	op := &textutil.DrawOptions{
		Options: textutil.Options{
			AutoWrap:         t.autoWrap,
			Face:             face,
			LineHeight:       t.lineHeight(context),
			HorizontalAlign:  textutil.HorizontalAlign(t.hAlign),
			VerticalAlign:    textutil.VerticalAlign(t.vAlign),
			TabWidth:         t.actualTabWidth(context),
			KeepTailingSpace: t.keepTailingSpace,
		},
		TextColor: textColor,
	}
	if start, end, ok := t.selectionToDraw(context); ok {
		if context.IsFocused(t) {
			op.DrawSelection = true
			op.SelectionStart = start
			op.SelectionEnd = end
			op.SelectionColor = draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.8)
		} else {
			op.DrawSelection = false
		}
	}
	if uStart, cStart, cEnd, uEnd, ok := t.compositionSelectionToDraw(context); ok {
		op.DrawComposition = true
		op.CompositionStart = uStart
		op.CompositionEnd = uEnd
		op.CompositionActiveStart = cStart
		op.CompositionActiveEnd = cEnd
		op.InactiveCompositionColor = draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.8)
		op.ActiveCompositionColor = draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.4)
		op.CompositionBorderWidth = float32(textCursorWidth(context))
	}
	textutil.Draw(textBounds, dst, t.textToDraw(context, true), op)
}

func (t *Text) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textSize(context, constraints, false)
}

func (t *Text) boldTextSize(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.textSize(context, constraints, true)
}

func (t *Text) textSize(context *guigui.Context, constraints guigui.Constraints, forceBold bool) image.Point {
	constraintWidth := math.MaxInt
	if w, ok := constraints.FixedWidth(); ok {
		constraintWidth = w
	}
	if constraintWidth == 0 {
		constraintWidth = 1
	}

	bold := t.bold || forceBold
	key := newTextSizeCacheKey(t.autoWrap, bold)
	for i, entry := range t.cachedTextSizes[key] {
		if entry.constraintWidth == 0 {
			continue
		}
		if entry.constraintWidth != constraintWidth {
			continue
		}
		// Move the used entry to the head.
		if i != 0 {
			copy(t.cachedTextSizes[key][1:i+1], t.cachedTextSizes[key][:i])
			t.cachedTextSizes[key][0] = entry
		}
		return entry.size
	}

	txt := t.textToDraw(context, true)
	w, h := textutil.Measure(constraintWidth, txt, t.autoWrap, t.face(context, bold), t.lineHeight(context), t.actualTabWidth(context), t.keepTailingSpace)
	// If width is 0, the text's bounds and visible bounds are empty, and nothing including its cursor is rendered.
	// Force to set a positive number as the width.
	w = max(w, 1)

	s := image.Pt(int(math.Ceil(w)), int(math.Ceil(h)))

	// Put the new entry at the head.
	copy(t.cachedTextSizes[key][1:], t.cachedTextSizes[key][:])
	t.cachedTextSizes[key][0] = cachedTextSizeEntry{
		constraintWidth: constraintWidth,
		size:            s,
	}

	return s
}

func (t *Text) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if t.selectable || t.editable {
		return ebiten.CursorShapeText, true
	}
	return 0, false
}

func (t *Text) cursorPosition(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (position textutil.TextPosition, ok bool) {
	if !context.IsFocused(t) {
		return textutil.TextPosition{}, false
	}
	if !t.editable {
		return textutil.TextPosition{}, false
	}
	start, end := t.field.Selection()
	if start < 0 {
		return textutil.TextPosition{}, false
	}
	if end < 0 {
		return textutil.TextPosition{}, false
	}

	_, e, ok := t.selectionToDraw(context)
	if !ok {
		return textutil.TextPosition{}, false
	}

	return t.textPosition(context, widgetBounds.Bounds(), e, true)
}

func (t *Text) textIndexFromPosition(context *guigui.Context, textBounds image.Rectangle, position image.Point, showComposition bool) int {
	textContentBounds := t.textContentBounds(context, textBounds)
	if position.Y < textContentBounds.Min.Y {
		return 0
	}
	txt := t.textToDraw(context, showComposition)
	if position.Y >= textContentBounds.Max.Y {
		return len(txt)
	}
	op := &textutil.Options{
		AutoWrap:         t.autoWrap,
		Face:             t.face(context, false),
		LineHeight:       t.lineHeight(context),
		HorizontalAlign:  textutil.HorizontalAlign(t.hAlign),
		VerticalAlign:    textutil.VerticalAlign(t.vAlign),
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}
	position = position.Sub(textContentBounds.Min)
	idx := textutil.TextIndexFromPosition(textContentBounds.Dx(), position, txt, op)
	if idx < 0 || idx > len(txt) {
		return -1
	}
	return idx
}

func (t *Text) textPosition(context *guigui.Context, bounds image.Rectangle, index int, showComposition bool) (position textutil.TextPosition, ok bool) {
	textBounds := t.textContentBounds(context, bounds)
	txt := t.textToDraw(context, showComposition)
	op := &textutil.Options{
		AutoWrap:         t.autoWrap,
		Face:             t.face(context, false),
		LineHeight:       t.lineHeight(context),
		HorizontalAlign:  textutil.HorizontalAlign(t.hAlign),
		VerticalAlign:    textutil.VerticalAlign(t.vAlign),
		TabWidth:         t.actualTabWidth(context),
		KeepTailingSpace: t.keepTailingSpace,
	}
	pos0, pos1, count := textutil.TextPositionFromIndex(textBounds.Dx(), txt, index, op)
	if count == 0 {
		return textutil.TextPosition{}, false
	}
	pos := pos0
	if count == 2 {
		pos = pos1
	}
	return textutil.TextPosition{
		X:      pos.X + float64(textBounds.Min.X),
		Top:    pos.Top + float64(textBounds.Min.Y),
		Bottom: pos.Bottom + float64(textBounds.Min.Y),
	}, true
}

func (t *Text) OnFocusChanged(context *guigui.Context, focused bool) {
	if !t.editable {
		return
	}
	if focused {
		t.field.Focus()
		t.cursor.resetCounter()
		start, end := t.field.Selection()
		if start < 0 || end < 0 {
			t.doSelectAll()
		}
	} else {
		t.commit()
	}
}

func textCursorWidth(context *guigui.Context) int {
	return int(2 * context.Scale())
}

func (t *Text) cursorBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	pos, ok := t.cursorPosition(context, widgetBounds)
	if !ok {
		return image.Rectangle{}
	}
	w := textCursorWidth(context)
	return image.Rect(int(pos.X)-w/2, int(pos.Top), int(pos.X)+w/2, int(pos.Bottom))
}

func (t *Text) adjustScrollOffset(context *guigui.Context, contentBounds image.Rectangle, textBounds image.Rectangle) (dx, dy float64) {
	start, end, ok := t.selectionToDraw(context)
	if !ok {
		return
	}
	if t.prevStart == start && t.prevEnd == end {
		return
	}
	t.prevStart = start
	t.prevEnd = end

	if pos, ok := t.textPosition(context, textBounds, end, true); ok {
		dx += min(float64(contentBounds.Max.X)-pos.X, 0)
		dy += min(float64(contentBounds.Max.Y)-pos.Bottom, 0)
	}
	if pos, ok := t.textPosition(context, textBounds, start, true); ok {
		dx += max(float64(contentBounds.Min.X)-pos.X, 0)
		dy += max(float64(contentBounds.Min.Y)-pos.Top, 0)
	}
	return dx, dy
}

func (t *Text) CanCut() bool {
	if !t.editable {
		return false
	}
	start, end := t.field.Selection()
	return start != end
}

func (t *Text) CanCopy() bool {
	start, end := t.field.Selection()
	return start != end
}

func (t *Text) CanPaste() bool {
	if !t.editable {
		return false
	}
	ct, err := clipboard.ReadAll()
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	return ct != ""
}

func (t *Text) Cut() bool {
	start, end := t.field.Selection()
	if start == end {
		return false
	}
	if err := clipboard.WriteAll(t.field.Text()[start:end]); err != nil {
		slog.Error(err.Error())
		return false
	}
	text := t.field.Text()[:start] + t.field.Text()[end:]
	t.setTextAndSelection(text, start, start, -1, true)
	return true
}

func (t *Text) Copy() bool {
	start, end := t.field.Selection()
	if start == end {
		return false
	}
	if err := clipboard.WriteAll(t.field.Text()[start:end]); err != nil {
		slog.Error(err.Error())
		return false
	}
	return true
}

func (t *Text) Paste() bool {
	start, end := t.field.Selection()
	ct, err := clipboard.ReadAll()
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	text := t.field.Text()[:start] + ct + t.field.Text()[end:]
	t.setTextAndSelection(text, start+len(ct), start+len(ct), -1, true)
	return true
}

type textCursor struct {
	guigui.DefaultWidget

	text *Text

	counter   int
	prevShown bool
	prevPos   textutil.TextPosition
	prevOK    bool
}

func (t *textCursor) resetCounter() {
	t.counter = 0
}

func (t *textCursor) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	pos, ok := t.text.cursorPosition(context, widgetBounds)
	if t.prevPos != pos {
		t.resetCounter()
	}
	t.prevPos = pos
	t.prevOK = ok

	t.counter++
	if r := t.shouldRenderCursor(context, widgetBounds, t.text); t.prevShown != r {
		t.prevShown = r
		// TODO: This is not efficient. Improve this.
		guigui.RequestRedraw(t)
	}
	return nil
}

func (t *textCursor) shouldRenderCursor(context *guigui.Context, widgetBounds *guigui.WidgetBounds, text *Text) bool {
	offset := ebiten.TPS() / 2
	if t.counter > offset && (t.counter-offset)%ebiten.TPS() >= ebiten.TPS()/2 {
		return false
	}
	if _, ok := text.cursorPosition(context, widgetBounds); !ok {
		return false
	}
	s, e, ok := text.selectionToDraw(context)
	if !ok {
		return false
	}
	if s != e {
		return false
	}
	return true
}

func (t *textCursor) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if !t.shouldRenderCursor(context, widgetBounds, t.text) {
		return
	}
	b := widgetBounds.Bounds()
	vector.FillRect(dst, float32(b.Min.X), float32(b.Min.Y), float32(b.Dx()), float32(b.Dy()), draw.Color(context.ColorMode(), draw.ColorTypeAccent, 0.4), false)
}

func replaceNewLinesWithSpace(text string, start, end, shiftIndex int) (string, int, int, int) {
	var buf strings.Builder
	for {
		pos, len := textutil.FirstLineBreakPositionAndLen(text)
		if len == 0 {
			buf.WriteString(text)
			break
		}
		buf.WriteString(text[:pos])
		origLen := buf.Len()
		buf.WriteString(" ")
		if diff := len - 1; diff > 0 {
			if origLen < start {
				if start >= origLen+len {
					start -= diff
				} else {
					// This is a very rare case, e.g. the position is in between '\r' and '\n'.
					start = origLen + 1
				}
			}
			if origLen < end {
				if end >= origLen+len {
					end -= diff
				} else {
					end = origLen + 1
				}
			}
			if shiftIndex >= 0 && origLen < shiftIndex {
				if shiftIndex >= origLen+len {
					shiftIndex -= diff
				} else {
					shiftIndex = origLen + 1
				}
			}
		}
		text = text[pos+len:]
	}
	text = buf.String()

	return text, start, end, shiftIndex
}
