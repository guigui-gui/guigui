// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type TooltipAreas struct {
	guigui.DefaultWidget

	button       basicwidget.Button
	text         basicwidget.Text
	selectWidget basicwidget.Select[int]
	tooltipArea1 basicwidget.TooltipArea
	tooltipArea2 basicwidget.TooltipArea
	tooltipArea3 basicwidget.TooltipArea

	popupButton  basicwidget.Button
	popup        basicwidget.Popup
	popupContent guigui.WidgetWithSize[*tooltipPopupContent]

	layoutItems     []guigui.LinearLayoutItem
	selectRowItems  []guigui.LinearLayoutItem
	selectRowLayout guigui.LinearLayout
	itemBoundsArr   []image.Rectangle
}

func (t *TooltipAreas) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.button)
	adder.AddWidget(&t.tooltipArea1)
	adder.AddWidget(&t.text)
	adder.AddWidget(&t.tooltipArea2)
	adder.AddWidget(&t.selectWidget)
	adder.AddWidget(&t.tooltipArea3)

	t.button.SetText("Hover me")
	t.tooltipArea1.SetText("This is a button tooltip")

	t.text.SetValue("Hover over this text to see a tooltip")
	t.tooltipArea2.SetText("This is a text tooltip")

	t.selectWidget.SetItemsByStrings([]string{
		"Apple",
		"Banana",
		"Cherry",
		"Date",
		"Elderberry",
		"Fig",
		"Grape",
		"Honeydew",
		"Kiwi",
		"Lemon",
	})
	if t.selectWidget.SelectedItemIndex() < 0 {
		t.selectWidget.SelectItemByIndex(0)
	}
	t.tooltipArea3.SetText("This is a select tooltip")

	adder.AddWidget(&t.popupButton)
	adder.AddWidget(&t.popup)

	t.popupButton.SetText("Show a popup with a tooltip")
	t.popupButton.OnUp(func(context *guigui.Context) {
		t.popup.SetOpen(true)
	})

	t.popupContent.Widget().SetPopup(&t.popup)
	t.popup.SetContent(&t.popupContent)
	t.popup.SetCloseByClickingOutside(true)
	t.popup.SetAnimated(true)
	t.popupContent.SetFixedSize(t.popupContentSize(context))

	return nil
}

func (t *TooltipAreas) popupContentSize(context *guigui.Context) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(int(12*u), int(6*u))
}

func (t *TooltipAreas) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)

	t.selectRowItems = slices.Delete(t.selectRowItems, 0, len(t.selectRowItems))
	t.selectRowItems = append(t.selectRowItems,
		guigui.LinearLayoutItem{
			Widget: &t.selectWidget,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
	)
	t.selectRowLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.selectRowItems,
	}

	t.layoutItems = slices.Delete(t.layoutItems, 0, len(t.layoutItems))
	t.layoutItems = append(t.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &t.button,
		},
		guigui.LinearLayoutItem{
			Widget: &t.text,
		},
		guigui.LinearLayoutItem{
			Layout: &t.selectRowLayout,
		},
		guigui.LinearLayoutItem{
			Widget: &t.popupButton,
		},
	)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     t.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}
}

func (t *TooltipAreas) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layout := t.layout(context)
	layout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	t.itemBoundsArr = layout.AppendItemBounds(t.itemBoundsArr[:0], context, widgetBounds.Bounds())
	layouter.LayoutWidget(&t.tooltipArea1, t.itemBoundsArr[0])
	layouter.LayoutWidget(&t.tooltipArea2, t.itemBoundsArr[1])
	selectBounds := t.selectRowLayout.ItemBoundsAt(0, context, t.itemBoundsArr[2])
	layouter.LayoutWidget(&t.tooltipArea3, selectBounds)

	appBounds := context.AppBounds()
	t.popup.SetBackgroundBounds(appBounds)
	contentSize := t.popupContentSize(context)
	center := image.Point{
		X: appBounds.Min.X + (appBounds.Dx()-contentSize.X)/2,
		Y: appBounds.Min.Y + (appBounds.Dy()-contentSize.Y)/2,
	}
	layouter.LayoutWidget(&t.popup, image.Rectangle{
		Min: center,
		Max: center.Add(contentSize),
	})
}

// tooltipPopupContent is the content of a popup that itself contains a tooltip area,
// demonstrating that a tooltip still works while another popup is open.
type tooltipPopupContent struct {
	guigui.DefaultWidget

	popup *basicwidget.Popup

	descriptionText basicwidget.Text
	button          basicwidget.Button
	tooltipArea     basicwidget.TooltipArea
	closeButton     basicwidget.Button

	buttonRowLayout guigui.LinearLayout
	buttonRowItems  []guigui.LinearLayoutItem
	layoutItems     []guigui.LinearLayoutItem
	itemBoundsArr   []image.Rectangle
}

func (s *tooltipPopupContent) SetPopup(popup *basicwidget.Popup) {
	s.popup = popup
}

func (s *tooltipPopupContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.descriptionText)
	adder.AddWidget(&s.button)
	adder.AddWidget(&s.tooltipArea)
	adder.AddWidget(&s.closeButton)

	s.descriptionText.SetMultiline(true)
	s.descriptionText.SetValue("Hover over the button to see a tooltip.")

	s.button.SetText("Hover me")
	s.tooltipArea.SetText("This is a tooltip inside a popup")

	s.closeButton.SetText("Close")
	s.closeButton.OnUp(func(context *guigui.Context) {
		s.popup.SetOpen(false)
	})

	return nil
}

func (s *tooltipPopupContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)

	s.buttonRowItems = slices.Delete(s.buttonRowItems, 0, len(s.buttonRowItems))
	s.buttonRowItems = append(s.buttonRowItems,
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &s.closeButton,
		},
	)
	s.buttonRowLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     s.buttonRowItems,
	}

	s.layoutItems = slices.Delete(s.layoutItems, 0, len(s.layoutItems))
	s.layoutItems = append(s.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &s.descriptionText,
		},
		guigui.LinearLayoutItem{
			Widget: &s.button,
		},
		guigui.LinearLayoutItem{
			Size: guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(s.closeButton.Measure(context, guigui.Constraints{}).Y),
			Layout: &s.buttonRowLayout,
		},
	)
	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     s.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}
	layout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)

	s.itemBoundsArr = layout.AppendItemBounds(s.itemBoundsArr[:0], context, widgetBounds.Bounds())
	layouter.LayoutWidget(&s.tooltipArea, s.itemBoundsArr[1])
}
