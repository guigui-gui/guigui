// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"
	"io"
	"log/slog"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

var (
	textEventKeyJustPressed guigui.EventKey = guigui.GenerateEventKey()
	textEventValueChanged   guigui.EventKey = guigui.GenerateEventKey()
	textEventScrollDelta    guigui.EventKey = guigui.GenerateEventKey()
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

	field             textinput.Field
	valueBuilder      stringBuilderWithRange
	valueEqualChecker stringEqualChecker
	nextTextSet       bool
	nextText          string
	nextSelectAll     bool
	textInited        bool

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

	cachedTextSizes       [4][4]cachedTextSizeEntry
	cachedDefaultTabWidth float64
	lastFace              text.Face
	lastScale             float64

	tmpLocales []language.Tag

	drawOptions textutil.DrawOptions

	prevStart              int
	prevEnd                int
	paddingForScrollOffset guigui.Padding

	onFocusChanged func(context *guigui.Context, focused bool)
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

func (t *Text) setOnScrollDelta(f func(context *guigui.Context, deltaX, deltaY float64)) {
	guigui.SetEventHandler(t, textEventScrollDelta, f)
}

func (t *Text) resetCachedTextSize() {
	clear(t.cachedTextSizes[:])
	t.cachedDefaultTabWidth = 0
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

	context.SetPassthrough(&t.cursor, true)

	if t.selectable || t.editable {
		t.cursor.text = t
	}

	if t.onFocusChanged == nil {
		t.onFocusChanged = func(context *guigui.Context, focused bool) {
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
	}
	guigui.SetOnFocusChanged(t, t.onFocusChanged)

	return nil
}

func (t *Text) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&t.cursor, t.cursorBounds(context, widgetBounds))
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
		t.setSelection(0, 0, -1, false)
	}
	guigui.RequestRebuild(t)
}

func (t *Text) isEqualToStringValue(text string) bool {
	t.valueEqualChecker.Reset(text)
	_ = t.field.WriteText(&t.valueEqualChecker)
	return t.valueEqualChecker.Result()
}

func (t *Text) stringValue() string {
	t.valueBuilder.Reset()
	_ = t.field.WriteText(&t.valueBuilder)
	return t.valueBuilder.String()
}

func (t *Text) stringValueWithRange(start, end int) string {
	t.valueBuilder.ResetWithRange(start, end)
	_ = t.field.WriteText(&t.valueBuilder)
	return t.valueBuilder.String()
}

func (t *Text) stringValueForRendering() string {
	t.valueBuilder.Reset()
	_ = t.field.WriteTextForRendering(&t.valueBuilder)
	return t.valueBuilder.String()
}

func (t *Text) Value() string {
	if t.nextTextSet {
		return t.nextText
	}
	return t.stringValue()
}

func (t *Text) SetValue(text string) {
	if t.nextTextSet && t.nextText == text {
		return
	}
	if !t.nextTextSet && t.isEqualToStringValue(text) {
		return
	}
	if !t.editable {
		t.setText(text, false)
		return
	}

	// Do not call t.setText here. Update the actual value later.
	// For example, when a user is editing, the text should not be changed.
	// Another case is that SetMultiline might be called later.
	t.nextText = text
	t.nextTextSet = true
	t.resetCachedTextSize()
}

func (t *Text) ForceSetValue(text string) {
	t.setText(text, false)
}

func (t *Text) ReplaceValueAtSelection(text string) {
	if text == "" {
		return
	}
	t.replaceTextAtSelection(text)
	t.resetCachedTextSize()
}

func (t *Text) CommitWithCurrentInputValue() {
	t.nextText = ""
	t.nextTextSet = false
	// Fire the event even if the text is not changed.
	guigui.DispatchEvent(t, textEventValueChanged, t.stringValue(), true)
}

func (t *Text) selectAll() {
	if t.nextTextSet {
		t.nextSelectAll = true
		return
	}
	t.doSelectAll()
}

func (t *Text) doSelectAll() {
	t.setSelection(0, t.field.TextLengthInBytes(), -1, false)
}

func (t *Text) setSelection(start, end int, shiftIndex int, adjustScroll bool) bool {
	t.selectionShiftIndexPlus1 = shiftIndex + 1
	if start > end {
		start, end = end, start
	}

	if s, e := t.field.Selection(); s == start && e == end {
		return false
	}
	t.field.SetSelection(start, end)
	guigui.RequestRebuild(t)

	if !adjustScroll {
		t.prevStart = start
		t.prevEnd = end
	}

	return true
}

func (t *Text) replaceTextAtSelection(text string) {
	start, end := t.field.Selection()
	t.replaceTextAt(text, start, end)
}

func (t *Text) replaceTextAt(text string, start, end int) {
	if !t.multiline {
		text, start, end = replaceNewLinesWithSpace(text, start, end)
	}

	t.selectionShiftIndexPlus1 = 0
	if start > end {
		start, end = end, start
	}
	if s, e := t.field.Selection(); text == t.stringValueWithRange(start, end) && s == start && e == end {
		return
	}
	t.field.ReplaceText(text, start, end)
	t.resetCachedTextSize()
	guigui.DispatchEvent(t, textEventValueChanged, t.stringValue(), false)

	t.nextText = ""
	t.nextTextSet = false
}

func (t *Text) setText(text string, selectAll bool) bool {
	if !t.multiline {
		text, _, _ = replaceNewLinesWithSpace(text, 0, 0)
	}

	t.selectionShiftIndexPlus1 = 0

	textChanged := !t.isEqualToStringValue(text)
	if s, e := t.field.Selection(); !textChanged && (!selectAll || s == 0 && e == len(text)) {
		return false
	}

	var start, end int
	if selectAll {
		end = len(text)
	}
	// When selectAll is false, the current selection range might be no longer valid.
	// Reset the selection to (0, 0).

	if textChanged {
		if t.textInited || t.field.TextLengthInBytes() > 0 {
			t.field.SetTextAndSelection(text, start, end)
		} else {
			// Reset the text so that the undo history's first item is the initial text.
			t.field.ResetText(text)
			t.field.SetSelection(start, end)
		}
		t.resetCachedTextSize()
		guigui.DispatchEvent(t, textEventValueChanged, t.stringValue(), false)
	} else {
		t.field.SetSelection(0, len(text))
	}
	guigui.RequestRebuild(t)

	// Do not adjust scroll.
	t.prevStart = start
	t.prevEnd = end
	t.nextText = ""
	t.nextTextSet = false
	t.textInited = true

	return true
}

func (t *Text) SetLocales(locales []language.Tag) {
	if slices.Equal(t.locales, locales) {
		return
	}

	t.locales = append([]language.Tag(nil), locales...)
	guigui.RequestRebuild(t)
}

func (t *Text) SetBold(bold bool) {
	if t.bold == bold {
		return
	}

	t.bold = bold
	guigui.RequestRebuild(t)
}

func (t *Text) SetTabular(tabular bool) {
	if t.tabular == tabular {
		return
	}

	t.tabular = tabular
	guigui.RequestRebuild(t)
}

func (t *Text) SetTabWidth(tabWidth float64) {
	if t.tabWidth == tabWidth {
		return
	}
	t.tabWidth = tabWidth
	t.resetCachedTextSize()
	guigui.RequestRebuild(t)
}

func (t *Text) actualTabWidth(context *guigui.Context) float64 {
	if t.tabWidth > 0 {
		return t.tabWidth
	}
	if t.cachedDefaultTabWidth > 0 {
		return t.cachedDefaultTabWidth
	}
	face := t.face(context, false)
	t.cachedDefaultTabWidth = text.Advance("        ", face)
	return t.cachedDefaultTabWidth
}

func (t *Text) scale() float64 {
	return t.scaleMinus1 + 1
}

func (t *Text) SetScale(scale float64) {
	if t.scaleMinus1 == scale-1 {
		return
	}

	t.scaleMinus1 = scale - 1
	guigui.RequestRebuild(t)
}

func (t *Text) HorizontalAlign() HorizontalAlign {
	return t.hAlign
}

func (t *Text) SetHorizontalAlign(align HorizontalAlign) {
	if t.hAlign == align {
		return
	}

	t.hAlign = align
	guigui.RequestRebuild(t)
}

func (t *Text) VerticalAlign() VerticalAlign {
	return t.vAlign
}

func (t *Text) SetVerticalAlign(align VerticalAlign) {
	if t.vAlign == align {
		return
	}

	t.vAlign = align
	guigui.RequestRebuild(t)
}

func (t *Text) SetColor(color color.Color) {
	if draw.EqualColor(t.color, color) {
		return
	}

	t.color = color
	guigui.RequestRebuild(t)
}

func (t *Text) SetOpacity(opacity float64) {
	if 1-t.transparent == opacity {
		return
	}

	t.transparent = 1 - opacity
	guigui.RequestRebuild(t)
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
	guigui.RequestRebuild(t)
}

func (t *Text) IsMultiline() bool {
	return t.multiline
}

func (t *Text) SetMultiline(multiline bool) {
	if t.multiline == multiline {
		return
	}

	t.multiline = multiline
	guigui.RequestRebuild(t)
}

func (t *Text) SetAutoWrap(autoWrap bool) {
	if t.autoWrap == autoWrap {
		return
	}

	t.autoWrap = autoWrap
	guigui.RequestRebuild(t)
}

func (t *Text) setKeepTailingSpace(keep bool) {
	if t.keepTailingSpace == keep {
		return
	}

	t.keepTailingSpace = keep
	guigui.RequestRebuild(t)
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
	return float64(LineHeight(context)) * (t.scaleMinus1 + 1)
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
			if t.setSelection(start, end, -1, true) {
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
			guigui.RequestRebuild(t)
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
				t.setSelection(idx, idx, -1, false)
			}
		}
	case 2:
		t.dragging = true
		start, end := findWordBoundaries(t.stringValue(), idx)
		t.selectionDragStartPlus1 = start + 1
		t.selectionDragEndPlus1 = end + 1
		t.setSelection(start, end, -1, false)
	case 3:
		t.doSelectAll()
	}

	context.SetFocused(t, true)

	t.lastClickTick = ebiten.Tick()
	t.lastClickTextIndex = idx
}

func (t *Text) textToDraw(context *guigui.Context, showComposition bool) string {
	if showComposition {
		return t.stringValueForRendering()
	}
	return t.stringValue()
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
		origText := t.stringValue()
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
			guigui.RequestRebuild(t)
			// Reset the cache size before adjust the scroll offset in order to get the correct text size.gi
			t.resetCachedTextSize()
			if !t.isEqualToStringValue(origText) {
				guigui.DispatchEvent(t, textEventValueChanged, t.stringValue(), false)
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
				t.replaceTextAtSelection("\n")
			} else {
				t.commit()
			}
			return guigui.HandleInputByWidget(t)
		case isKeyRepeating(ebiten.KeyBackspace) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyH):
			start, end := t.field.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if start > 0 {
				pos := textutil.PrevPositionOnGraphemes(t.stringValue(), start)
				t.replaceTextAt("", pos, start)
			}
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyD) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyD):
			// Delete
			start, end := t.field.Selection()
			if start != end {
				t.replaceTextAtSelection("")
			} else if isDarwin() && end < t.field.TextLengthInBytes() {
				pos := textutil.NextPositionOnGraphemes(t.stringValue(), end)
				t.replaceTextAt("", start, pos)
			}
			return guigui.HandleInputByWidget(t)
		case isKeyRepeating(ebiten.KeyDelete):
			// Delete one cluster
			if _, end := t.field.Selection(); end < t.field.TextLengthInBytes() {
				pos := textutil.NextPositionOnGraphemes(t.stringValue(), end)
				t.replaceTextAt("", start, pos)
			}
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyX) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyX):
			t.Cut()
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyV) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyV):
			t.Paste()
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyY) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && ebiten.IsKeyPressed(ebiten.KeyShift) && isKeyRepeating(ebiten.KeyZ):
			t.Redo()
			return guigui.HandleInputByWidget(t)
		case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyZ) ||
			isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyZ):
			t.Undo()
			return guigui.HandleInputByWidget(t)
		}
	}

	switch {
	case isKeyRepeating(ebiten.KeyLeft) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyB):
		start, end := t.field.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.selectionShiftIndexPlus1-1 == end {
				pos := textutil.PrevPositionOnGraphemes(t.stringValue(), end)
				t.setSelection(start, pos, pos, true)
			} else {
				pos := textutil.PrevPositionOnGraphemes(t.stringValue(), start)
				t.setSelection(pos, end, pos, true)
			}
		} else {
			if start != end {
				t.setSelection(start, start, -1, true)
			} else if start > 0 {
				pos := textutil.PrevPositionOnGraphemes(t.stringValue(), start)
				t.setSelection(pos, pos, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyRight) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyF):
		start, end := t.field.Selection()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			if t.selectionShiftIndexPlus1-1 == start {
				pos := textutil.NextPositionOnGraphemes(t.stringValue(), start)
				t.setSelection(pos, end, pos, true)
			} else {
				pos := textutil.NextPositionOnGraphemes(t.stringValue(), end)
				t.setSelection(start, pos, pos, true)
			}
		} else {
			if start != end {
				t.setSelection(end, end, -1, true)
			} else if start < t.field.TextLengthInBytes() {
				pos := textutil.NextPositionOnGraphemes(t.stringValue(), start)
				t.setSelection(pos, pos, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyUp) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyP):
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
					t.setSelection(start, idx, idx, true)
				} else {
					t.setSelection(idx, end, idx, true)
				}
			} else {
				t.setSelection(idx, idx, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isKeyRepeating(ebiten.KeyDown) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyN):
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
					t.setSelection(idx, end, idx, true)
				} else {
					t.setSelection(start, idx, idx, true)
				}
			} else {
				t.setSelection(idx, idx, -1, true)
			}
		}
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyA):
		idx := 0
		start, end := t.field.Selection()
		if i := strings.LastIndex(t.stringValueWithRange(0, start), "\n"); i >= 0 {
			idx = i + 1
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(idx, end, idx, true)
		} else {
			t.setSelection(idx, idx, -1, true)
		}
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyE):
		idx := t.field.TextLengthInBytes()
		start, end := t.field.Selection()
		if i := strings.Index(t.stringValueWithRange(end, -1), "\n"); i >= 0 {
			idx = end + i
		}
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			t.setSelection(start, idx, idx, true)
		} else {
			t.setSelection(idx, idx, -1, true)
		}
		return guigui.HandleInputByWidget(t)
	case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyA) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyA):
		t.doSelectAll()
		return guigui.HandleInputByWidget(t)
	case !isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyC) ||
		isDarwin() && ebiten.IsKeyPressed(ebiten.KeyMeta) && isKeyRepeating(ebiten.KeyC):
		// Copy
		t.Copy()
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyK):
		// 'Kill' the text after the cursor or the selection.
		start, end := t.field.Selection()
		if start == end {
			end = strings.Index(t.stringValueWithRange(start, -1), "\n")
			if end < 0 {
				end = t.field.TextLengthInBytes()
			} else if end == 0 {
				end += start + 1
			} else {
				end += start
			}
		}
		t.tmpClipboard = t.stringValueWithRange(start, end)
		t.replaceTextAt("", start, end)
		return guigui.HandleInputByWidget(t)
	case isDarwin() && ebiten.IsKeyPressed(ebiten.KeyControl) && isKeyRepeating(ebiten.KeyY):
		// 'Yank' the killed text.
		if t.tmpClipboard != "" {
			t.replaceTextAtSelection(t.tmpClipboard)
		}
		return guigui.HandleInputByWidget(t)
	}

	return guigui.HandleInputResult{}
}

func (t *Text) commit() {
	guigui.DispatchEvent(t, textEventValueChanged, t.stringValue(), true)
	t.nextText = ""
	t.nextTextSet = false
}

func (t *Text) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	// Once a text is input, it is regarded as initialized.
	if t.field.TextLengthInBytes() > 0 {
		t.textInited = true
	}
	if (!t.editable || !context.IsFocused(t)) && t.nextTextSet {
		t.setText(t.nextText, t.nextSelectAll)
		t.nextSelectAll = false
	}

	// Adjust the scroll offset at Tick, as this requires all the widgets are already laid out.
	// TODO: The cursor position might be unstable when the text horizontal align is center or right. Fix this.
	if dx, dy := t.adjustScrollOffset(context, widgetBounds); dx != 0 || dy != 0 {
		guigui.DispatchEvent(t, textEventScrollDelta, dx, dy)
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
	op := &t.drawOptions
	op.Options.AutoWrap = t.autoWrap
	op.Options.Face = face
	op.Options.LineHeight = t.lineHeight(context)
	op.Options.HorizontalAlign = textutil.HorizontalAlign(t.hAlign)
	op.Options.VerticalAlign = textutil.VerticalAlign(t.vAlign)
	op.Options.TabWidth = t.actualTabWidth(context)
	op.Options.KeepTailingSpace = t.keepTailingSpace
	op.TextColor = textColor
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
	} else {
		op.DrawComposition = false
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
	for i := range t.cachedTextSizes[key] {
		// Use a pointer to avoid runtime.duffcopy.
		entry := &t.cachedTextSizes[key][i]
		if entry.constraintWidth == 0 {
			continue
		}
		if entry.constraintWidth != constraintWidth {
			continue
		}
		if i == 0 {
			return entry.size
		}

		// Move the used entry to the head.
		e := *entry
		copy(t.cachedTextSizes[key][1:i+1], t.cachedTextSizes[key][:i])
		t.cachedTextSizes[key][0] = e
		return e.size
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

func textCursorWidth(context *guigui.Context) int {
	return int(2 * context.Scale())
}

func (t *Text) cursorBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	pos, ok := t.cursorPosition(context, widgetBounds)
	if !ok {
		return image.Rectangle{}
	}
	w := textCursorWidth(context)
	paddingTop := 2 * t.scale() * context.Scale()
	paddingBottom := 1 * t.scale() * context.Scale()
	return image.Rect(int(pos.X)-w/2, int(pos.Top+paddingTop), int(pos.X)+w/2, int(pos.Bottom-paddingBottom))
}

func (t *Text) setPaddingForScrollOffset(padding guigui.Padding) {
	if t.paddingForScrollOffset == padding {
		return
	}
	t.paddingForScrollOffset = padding
	guigui.RequestRebuild(t)
}

func (t *Text) adjustScrollOffset(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (dx, dy float64) {
	start, end, ok := t.selectionToDraw(context)
	if !ok {
		return
	}
	if t.prevStart == start && t.prevEnd == end && !t.dragging {
		return
	}
	t.prevStart = start
	t.prevEnd = end

	textBounds := widgetBounds.Bounds()
	textVisibleBounds := widgetBounds.VisibleBounds()

	cx, cy := ebiten.CursorPosition()
	if pos, ok := t.textPosition(context, textBounds, end, true); ok {
		var deltaX, deltaY float64
		if t.dragging {
			deltaX = float64(textVisibleBounds.Max.X) - float64(cx) - float64(t.paddingForScrollOffset.End)
			deltaY = float64(textVisibleBounds.Max.Y) - float64(cy) - float64(t.paddingForScrollOffset.Bottom)
			if cx > textVisibleBounds.Max.X {
				deltaX /= 4
			} else {
				deltaX = 0
			}
			if cy > textVisibleBounds.Max.Y {
				deltaY /= 4
			} else {
				deltaY = 0
			}
		} else {
			deltaX = float64(textVisibleBounds.Max.X) - pos.X - float64(t.paddingForScrollOffset.End)
			deltaY = float64(textVisibleBounds.Max.Y) - pos.Bottom - float64(t.paddingForScrollOffset.Bottom)
		}
		deltaX = min(deltaX, 0)
		deltaY = min(deltaY, 0)
		dx += deltaX
		dy += deltaY
	}
	if pos, ok := t.textPosition(context, textBounds, start, true); ok {
		var deltaX, deltaY float64
		if t.dragging {
			deltaX = float64(textVisibleBounds.Min.X) - float64(cx) + float64(t.paddingForScrollOffset.Start)
			deltaY = float64(textVisibleBounds.Min.Y) - float64(cy) + float64(t.paddingForScrollOffset.Top)
			if cx < textVisibleBounds.Min.X {
				deltaX /= 4
			} else {
				deltaX = 0
			}
			if cy < textVisibleBounds.Min.Y {
				deltaY /= 4
			} else {
				deltaY = 0
			}
		} else {
			deltaX = float64(textVisibleBounds.Min.X) - pos.X + float64(t.paddingForScrollOffset.Start)
			deltaY = float64(textVisibleBounds.Min.Y) - pos.Top + float64(t.paddingForScrollOffset.Top)
		}
		deltaX = max(deltaX, 0)
		deltaY = max(deltaY, 0)
		dx += deltaX
		dy += deltaY
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

func (t *Text) CanUndo() bool {
	if !t.editable {
		return false
	}
	return t.field.CanUndo()
}

func (t *Text) CanRedo() bool {
	if !t.editable {
		return false
	}
	return t.field.CanRedo()
}

func (t *Text) Cut() bool {
	start, end := t.field.Selection()
	if start == end {
		return false
	}
	if err := clipboard.WriteAll(t.stringValueWithRange(start, end)); err != nil {
		slog.Error(err.Error())
		return false
	}
	t.replaceTextAtSelection("")
	return true
}

func (t *Text) Copy() bool {
	start, end := t.field.Selection()
	if start == end {
		return false
	}
	if err := clipboard.WriteAll(t.stringValueWithRange(start, end)); err != nil {
		slog.Error(err.Error())
		return false
	}
	return true
}

func (t *Text) Paste() bool {
	ct, err := clipboard.ReadAll()
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	t.replaceTextAtSelection(ct)
	return true
}

func (t *Text) Undo() bool {
	if !t.field.CanUndo() {
		return false
	}
	t.field.Undo()
	t.resetCachedTextSize()
	guigui.RequestRebuild(t)
	return true
}

func (t *Text) Redo() bool {
	if !t.field.CanRedo() {
		return false
	}
	t.field.Redo()
	t.resetCachedTextSize()
	guigui.RequestRebuild(t)
	return true
}

type textCursor struct {
	guigui.DefaultWidget

	text *Text

	counter   int
	prevAlpha float64
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
	if a := t.alpha(context, widgetBounds, t.text); t.prevAlpha != a {
		t.prevAlpha = a
		guigui.RequestRedraw(t)
	}
	return nil
}

func (t *textCursor) alpha(context *guigui.Context, widgetBounds *guigui.WidgetBounds, text *Text) float64 {
	if _, ok := text.cursorPosition(context, widgetBounds); !ok {
		return 0
	}
	s, e, ok := text.selectionToDraw(context)
	if !ok {
		return 0
	}
	if s != e {
		return 0
	}
	offset := ebiten.TPS() / 2
	if t.counter <= offset {
		return 1
	}
	interval := ebiten.TPS()
	c := (t.counter - offset) % interval
	if c < interval/5 {
		return 1 - float64(c)/float64(interval/5)
	}
	if c < interval*2/5 {
		return 0
	}
	if c < interval*3/5 {
		return float64(c-interval*2/5) / float64(interval/5)
	}
	return 1
}

func (t *textCursor) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	alpha := t.alpha(context, widgetBounds, t.text)
	if alpha == 0 {
		return
	}
	b := widgetBounds.Bounds()
	clr := draw.ScaleAlpha(draw.Color2(context.ColorMode(), draw.ColorTypeAccent, 0.5, 0.6), alpha)
	basicwidgetdraw.DrawRoundedRect(context, dst, b, clr, b.Dx()/2)
}

func replaceNewLinesWithSpace(text string, start, end int) (string, int, int) {
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
		}
		text = text[pos+len:]
	}
	text = buf.String()

	return text, start, end
}

type stringBuilderWithRange struct {
	buf      []byte
	start    int
	endPlus1 int
	offset   int
}

func (s *stringBuilderWithRange) Reset() {
	s.buf = s.buf[:0]
	s.start = 0
	s.endPlus1 = 0
	s.offset = 0
}

func (s *stringBuilderWithRange) ResetWithRange(start, end int) {
	s.buf = s.buf[:0]
	s.start = start
	s.endPlus1 = end + 1
	s.offset = 0
}

func (s *stringBuilderWithRange) Write(b []byte) (int, error) {
	origN := len(b)
	defer func() {
		s.offset += origN
	}()

	start := s.start
	end := math.MaxInt
	if s.endPlus1 > 0 {
		end = s.endPlus1 - 1
	}

	// Calculate the intersection of [s.offset, s.offset+len(b)) and [start, end).
	idx0 := max(s.offset, start)
	idx1 := min(s.offset+len(b), end)

	if idx0 >= idx1 {
		return origN, nil
	}

	s.buf = append(s.buf, b[idx0-s.offset:idx1-s.offset]...)
	return origN, nil
}

func (s *stringBuilderWithRange) String() string {
	return string(s.buf)
}

type stringEqualChecker struct {
	str    string
	pos    int
	result bool
}

func (s *stringEqualChecker) Reset(str string) {
	s.str = str
	s.pos = 0
	s.result = true
}

func (s *stringEqualChecker) Result() bool {
	if s.pos != len(s.str) {
		return false
	}
	return s.result
}

func (s *stringEqualChecker) Write(b []byte) (int, error) {
	if s.pos+len(b) > len(s.str) {
		s.result = false
		return 0, io.EOF
	}

	ss := s.str[s.pos : s.pos+len(b)]
	// Eliminates the boundary check.
	_ = ss[:len(b)]
	for i, v := range b {
		if ss[i] != v {
			s.result = false
			return i, io.EOF
		}
		s.pos++
	}
	return len(b), nil
}
