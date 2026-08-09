// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"image/color"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

var (
	toolbarEventBold              = guigui.GenerateEventKey()
	toolbarEventItalic            = guigui.GenerateEventKey()
	toolbarEventUnderline         = guigui.GenerateEventKey()
	toolbarEventStrikethrough     = guigui.GenerateEventKey()
	toolbarEventTextColorSelected = guigui.GenerateEventKey()
	toolbarEventHighlightSelected = guigui.GenerateEventKey()
	toolbarEventClear             = guigui.GenerateEventKey()
	toolbarEventScaleUp           = guigui.GenerateEventKey()
	toolbarEventScaleDown         = guigui.GenerateEventKey()
	toolbarEventUndo              = guigui.GenerateEventKey()
	toolbarEventRedo              = guigui.GenerateEventKey()
)

// Toolbar is the row of style buttons above the editor. The lit states and
// the undo/redo enabled states are pushed in by the owner; button presses
// and palette selections are reported through the On* handlers.
type Toolbar struct {
	guigui.DefaultWidget

	boldButton          basicwidget.Button
	italicButton        basicwidget.Button
	underlineButton     basicwidget.Button
	strikethroughButton basicwidget.Button
	textColorButton     basicwidget.Button
	highlightButton     basicwidget.Button
	scaleUpButton       basicwidget.Button
	scaleDownButton     basicwidget.Button
	clearButton         basicwidget.Button
	undoButton          basicwidget.Button
	redoButton          basicwidget.Button

	textColorPopup        basicwidget.Popup
	textColorPopupContent palettePopupContent
	highlightPopup        basicwidget.Popup
	highlightPopupContent palettePopupContent

	boldLit          bool
	italicLit        bool
	underlineLit     bool
	strikethroughLit bool
	canUndo          bool
	canRedo          bool

	// textColorButtonIndex and highlightButtonIndex locate the palette
	// buttons in the row layout for anchoring their popups.
	textColorButtonIndex int
	highlightButtonIndex int

	layoutItems []guigui.LinearLayoutItem
}

// textPalette is the fixed palette of the text color popup, from the color
// universal design palette.
var textPalette = []color.Color{
	color.RGBA{R: 0xff, G: 0x4b, B: 0x00, A: 0xff}, // Red.
	color.RGBA{R: 0xf6, G: 0xaa, B: 0x00, A: 0xff}, // Orange.
	color.RGBA{R: 0x03, G: 0xaf, B: 0x7a, A: 0xff}, // Green.
	color.RGBA{R: 0x00, G: 0x5a, B: 0xff, A: 0xff}, // Blue.
}

// highlightPalette is the fixed palette of the highlight popup: the text
// palette's colors with a translucent alpha.
var highlightPalette = translucentColors(textPalette)

// translucentColors returns colors with the alpha replaced by a translucent
// value.
func translucentColors(colors []color.Color) []color.Color {
	result := make([]color.Color, len(colors))
	for i, clr := range colors {
		r, g, b, _ := clr.RGBA()
		result[i] = color.NRGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: 0x40,
		}
	}
	return result
}

func (t *Toolbar) SetBoldLit(lit bool) {
	t.boldLit = lit
}

func (t *Toolbar) SetItalicLit(lit bool) {
	t.italicLit = lit
}

func (t *Toolbar) SetUnderlineLit(lit bool) {
	t.underlineLit = lit
}

func (t *Toolbar) SetStrikethroughLit(lit bool) {
	t.strikethroughLit = lit
}

func (t *Toolbar) SetCanUndo(canUndo bool) {
	t.canUndo = canUndo
}

func (t *Toolbar) SetCanRedo(canRedo bool) {
	t.canRedo = canRedo
}

func (t *Toolbar) OnBold(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventBold, f)
}

func (t *Toolbar) OnItalic(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventItalic, f)
}

func (t *Toolbar) OnUnderline(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventUnderline, f)
}

func (t *Toolbar) OnStrikethrough(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventStrikethrough, f)
}

// OnTextColorSelected sets the handler invoked when a text color popup entry
// is chosen. ok is false for the default entry, which clears the property.
func (t *Toolbar) OnTextColorSelected(f func(context *guigui.Context, clr color.Color, ok bool)) {
	guigui.SetEventHandler(t, toolbarEventTextColorSelected, f)
}

// OnHighlightSelected sets the handler invoked when a highlight popup entry
// is chosen. ok is false for the default entry, which clears the property.
func (t *Toolbar) OnHighlightSelected(f func(context *guigui.Context, clr color.Color, ok bool)) {
	guigui.SetEventHandler(t, toolbarEventHighlightSelected, f)
}

func (t *Toolbar) OnClear(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventClear, f)
}

func (t *Toolbar) OnScaleUp(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventScaleUp, f)
}

func (t *Toolbar) OnScaleDown(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventScaleDown, f)
}

func (t *Toolbar) OnUndo(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventUndo, f)
}

func (t *Toolbar) OnRedo(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, toolbarEventRedo, f)
}

// configureToggleButton applies the shared appearance and action of a
// toggle button: the icon, the lit look, and the event dispatched on press.
func (t *Toolbar) configureToggleButton(context *guigui.Context, button *basicwidget.Button, iconName string, lit bool, eventKey guigui.EventKey) error {
	img, err := theImageLoader.MonochromeImage(iconName, context.ColorMode())
	if err != nil {
		return err
	}
	button.SetIcon(img)
	button.SetToggleable(true)
	button.SetPressed(lit)
	button.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(t, eventKey)
	})
	return nil
}

func (t *Toolbar) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.boldButton)
	adder.AddWidget(&t.italicButton)
	adder.AddWidget(&t.underlineButton)
	adder.AddWidget(&t.strikethroughButton)
	adder.AddWidget(&t.textColorButton)
	adder.AddWidget(&t.highlightButton)
	adder.AddWidget(&t.scaleUpButton)
	adder.AddWidget(&t.scaleDownButton)
	adder.AddWidget(&t.clearButton)
	adder.AddWidget(&t.undoButton)
	adder.AddWidget(&t.redoButton)
	adder.AddWidget(&t.textColorPopup)
	adder.AddWidget(&t.highlightPopup)

	cm := context.ColorMode()

	if err := t.configureToggleButton(context, &t.boldButton, "format_bold", t.boldLit, toolbarEventBold); err != nil {
		return err
	}
	if err := t.configureToggleButton(context, &t.italicButton, "format_italic", t.italicLit, toolbarEventItalic); err != nil {
		return err
	}
	if err := t.configureToggleButton(context, &t.underlineButton, "format_underlined", t.underlineLit, toolbarEventUnderline); err != nil {
		return err
	}
	if err := t.configureToggleButton(context, &t.strikethroughButton, "strikethrough_s", t.strikethroughLit, toolbarEventStrikethrough); err != nil {
		return err
	}

	// Adjacent buttons of a group touch each other, so the corners on the
	// touching sides are sharp and only the group's outer corners stay
	// rounded.
	sharpEnd := basicwidget.Corners{TopEnd: true, BottomEnd: true}
	sharpStart := basicwidget.Corners{TopStart: true, BottomStart: true}
	sharpBoth := basicwidget.Corners{TopStart: true, TopEnd: true, BottomStart: true, BottomEnd: true}
	t.boldButton.SetSharpCorners(sharpEnd)
	t.italicButton.SetSharpCorners(sharpBoth)
	t.underlineButton.SetSharpCorners(sharpBoth)
	t.strikethroughButton.SetSharpCorners(sharpStart)
	t.textColorButton.SetSharpCorners(sharpEnd)
	t.highlightButton.SetSharpCorners(sharpStart)
	t.scaleUpButton.SetSharpCorners(sharpEnd)
	t.scaleDownButton.SetSharpCorners(sharpStart)
	t.undoButton.SetSharpCorners(sharpEnd)
	t.redoButton.SetSharpCorners(sharpStart)

	textColorImg, err := theImageLoader.MonochromeImage("format_color_text", cm)
	if err != nil {
		return err
	}
	highlightImg, err := theImageLoader.MonochromeImage("format_ink_highlighter", cm)
	if err != nil {
		return err
	}
	scaleUpImg, err := theImageLoader.MonochromeImage("text_increase", cm)
	if err != nil {
		return err
	}
	scaleDownImg, err := theImageLoader.MonochromeImage("text_decrease", cm)
	if err != nil {
		return err
	}
	clearImg, err := theImageLoader.MonochromeImage("format_clear", cm)
	if err != nil {
		return err
	}
	undoImg, err := theImageLoader.MonochromeImage("undo", cm)
	if err != nil {
		return err
	}
	redoImg, err := theImageLoader.MonochromeImage("redo", cm)
	if err != nil {
		return err
	}

	t.textColorButton.SetIcon(textColorImg)
	t.textColorButton.OnDown(func(context *guigui.Context) {
		t.textColorPopup.SetOpen(true)
	})
	t.highlightButton.SetIcon(highlightImg)
	t.highlightButton.OnDown(func(context *guigui.Context) {
		t.highlightPopup.SetOpen(true)
	})

	t.scaleUpButton.SetIcon(scaleUpImg)
	t.scaleUpButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(t, toolbarEventScaleUp)
	})
	t.scaleDownButton.SetIcon(scaleDownImg)
	t.scaleDownButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(t, toolbarEventScaleDown)
	})

	t.clearButton.SetIcon(clearImg)
	t.clearButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(t, toolbarEventClear)
	})

	t.undoButton.SetIcon(undoImg)
	context.SetEnabled(&t.undoButton, t.canUndo)
	t.undoButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(t, toolbarEventUndo)
	})
	t.redoButton.SetIcon(redoImg)
	context.SetEnabled(&t.redoButton, t.canRedo)
	t.redoButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(t, toolbarEventRedo)
	})

	t.textColorPopupContent.SetColors(textPalette)
	t.textColorPopupContent.OnSelected(func(context *guigui.Context, clr color.Color, ok bool) {
		t.textColorPopup.SetOpen(false)
		guigui.DispatchEvent(t, toolbarEventTextColorSelected, clr, ok)
	})
	t.textColorPopup.SetContent(&t.textColorPopupContent)
	t.textColorPopup.SetCloseByClickingOutside(true)
	t.textColorPopup.SetAnimated(true)

	t.highlightPopupContent.SetColors(highlightPalette)
	t.highlightPopupContent.OnSelected(func(context *guigui.Context, clr color.Color, ok bool) {
		t.highlightPopup.SetOpen(false)
		guigui.DispatchEvent(t, toolbarEventHighlightSelected, clr, ok)
	})
	t.highlightPopup.SetContent(&t.highlightPopupContent)
	t.highlightPopup.SetCloseByClickingOutside(true)
	t.highlightPopup.SetAnimated(true)

	return nil
}

func (t *Toolbar) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)
	buttonWidth := u

	t.layoutItems = slices.Delete(t.layoutItems, 0, len(t.layoutItems))
	appendButton := func(button *basicwidget.Button) int {
		t.layoutItems = append(t.layoutItems, guigui.LinearLayoutItem{
			Widget: button,
			Size:   guigui.FixedSize(buttonWidth),
		})
		return len(t.layoutItems) - 1
	}
	appendGroupGap := func() {
		t.layoutItems = append(t.layoutItems, guigui.LinearLayoutItem{
			Size: guigui.FixedSize(u / 2),
		})
	}

	appendButton(&t.undoButton)
	appendButton(&t.redoButton)
	appendGroupGap()
	appendButton(&t.boldButton)
	appendButton(&t.italicButton)
	appendButton(&t.underlineButton)
	appendButton(&t.strikethroughButton)
	appendGroupGap()
	t.textColorButtonIndex = appendButton(&t.textColorButton)
	t.highlightButtonIndex = appendButton(&t.highlightButton)
	appendGroupGap()
	appendButton(&t.scaleUpButton)
	appendButton(&t.scaleDownButton)
	appendGroupGap()
	appendButton(&t.clearButton)

	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.layoutItems,
		Padding: guigui.Padding{
			Start:  u / 4,
			Top:    u / 4,
			End:    u / 4,
			Bottom: u / 4,
		},
	}
}

func (t *Toolbar) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layout := t.layout(context)
	bounds := widgetBounds.Bounds()
	layout.LayoutWidgets(context, bounds, layouter)

	textColorButtonBounds := layout.ItemBoundsAt(t.textColorButtonIndex, context, bounds)
	layouter.LayoutWidget(&t.textColorPopup, t.popupBounds(context, &t.textColorPopupContent, textColorButtonBounds))
	highlightButtonBounds := layout.ItemBoundsAt(t.highlightButtonIndex, context, bounds)
	layouter.LayoutWidget(&t.highlightPopup, t.popupBounds(context, &t.highlightPopupContent, highlightButtonBounds))
}

// popupBounds returns the bounds of a palette popup anchored below its
// button, kept within the app bounds.
func (t *Toolbar) popupBounds(context *guigui.Context, content *palettePopupContent, buttonBounds image.Rectangle) image.Rectangle {
	u := basicwidget.UnitSize(context)
	size := content.Measure(context, guigui.Constraints{})
	pt := image.Pt(buttonBounds.Min.X, buttonBounds.Max.Y+u/8)
	appBounds := context.AppBounds()
	pt.X = min(pt.X, appBounds.Max.X-size.X)
	pt.X = max(pt.X, appBounds.Min.X)
	return image.Rectangle{
		Min: pt,
		Max: pt.Add(size),
	}
}

func (t *Toolbar) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return t.layout(context).Measure(context, constraints)
}
