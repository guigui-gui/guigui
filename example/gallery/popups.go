// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Popups struct {
	guigui.DefaultWidget

	forms                        [2]basicwidget.Form
	darkenBackgroundText         basicwidget.Text
	darkenBackgroundToggle       basicwidget.Toggle
	blurBackgroundText           basicwidget.Text
	blurBackgroundToggle         basicwidget.Toggle
	closeByClickingOutsideText   basicwidget.Text
	closeByClickingOutsideToggle basicwidget.Toggle
	narrowBackgroundText         basicwidget.Text
	narrowBackgroundToggle       basicwidget.Toggle
	modalText                    basicwidget.Text
	modalToggle                  basicwidget.Toggle
	showButton                   basicwidget.Button

	contextMenuPopupText          basicwidget.Text
	contextMenuPopupClickHereText popupClickHereText

	simplePopup        basicwidget.Popup
	simplePopupContent guigui.WidgetWithSize[*simplePopupContent]

	contextMenuPopup basicwidget.PopupMenu[int]

	contextMenuPopupPosition image.Point

	layoutItems []guigui.LinearLayoutItem
}

func (p *Popups) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	v, ok := context.Env(p, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

	for i := range p.forms {
		adder.AddWidget(&p.forms[i])
	}
	adder.AddWidget(&p.simplePopup)
	adder.AddWidget(&p.contextMenuPopup)

	p.darkenBackgroundText.SetValue("Darken the background")
	p.blurBackgroundText.SetValue("Blur the background")
	p.closeByClickingOutsideText.SetValue("Close by clicking outside")
	p.narrowBackgroundText.SetValue("Narrow the background")
	p.modalText.SetValue("Modal")
	p.showButton.SetText("Show")
	p.showButton.OnUp(func(context *guigui.Context) {
		p.simplePopup.SetOpen(true)
	})

	p.forms[0].SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &p.darkenBackgroundText,
			SecondaryWidget: &p.darkenBackgroundToggle,
		},
		{
			PrimaryWidget:   &p.blurBackgroundText,
			SecondaryWidget: &p.blurBackgroundToggle,
		},
		{
			PrimaryWidget:   &p.closeByClickingOutsideText,
			SecondaryWidget: &p.closeByClickingOutsideToggle,
		},
		{
			PrimaryWidget:   &p.narrowBackgroundText,
			SecondaryWidget: &p.narrowBackgroundToggle,
		},
		{
			PrimaryWidget:   &p.modalText,
			SecondaryWidget: &p.modalToggle,
		},
		{
			SecondaryWidget: &p.showButton,
		},
	})

	p.contextMenuPopupText.SetValue("Context menu")
	p.contextMenuPopupClickHereText.Text().SetValue("Click here by the right button")

	p.forms[1].SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &p.contextMenuPopupText,
			SecondaryWidget: &p.contextMenuPopupClickHereText,
		},
	})

	p.simplePopupContent.Widget().SetPopup(&p.simplePopup)
	p.simplePopup.SetContent(&p.simplePopupContent)
	p.simplePopup.SetBackgroundDark(p.darkenBackgroundToggle.Value())
	p.simplePopup.SetBackgroundBlurred(p.blurBackgroundToggle.Value())
	p.simplePopup.SetCloseByClickingOutside(p.closeByClickingOutsideToggle.Value())
	p.modalToggle.OnValueChanged(func(context *guigui.Context, modal bool) {
		model.Popups().SetModal(modal)
	})
	p.modalToggle.SetValue(model.Popups().Modal())
	p.simplePopup.SetModal(model.Popups().Modal())
	p.simplePopup.SetAnimated(true)

	p.simplePopupContent.SetFixedSize(p.contentSize(context))

	p.contextMenuPopup.SetItems(
		[]basicwidget.PopupMenuItem[int]{
			{
				Text:    "Item 1",
				KeyText: "Foo",
			},
			{
				Text:    "Item 2",
				KeyText: "Bar",
			},
			{
				Text: "Item 3",
			},
			{
				Border: true,
			},
			{
				Text:     "Item 4",
				Disabled: true,
			},
		},
	)
	// A context menu's position is updated at HandlePointingInput.

	p.contextMenuPopupClickHereText.OnClicked(func(context *guigui.Context, pt image.Point) {
		p.contextMenuPopupPosition = pt
		p.contextMenuPopup.SetOpen(true)
	})

	return nil
}

func (p *Popups) contentSize(context *guigui.Context) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(int(12*u), int(6*u))
}

func (p *Popups) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	appBounds := context.AppBounds()

	popupBounds := appBounds
	if p.narrowBackgroundToggle.Value() {
		popupBounds = widgetBounds.Bounds()
	}
	p.simplePopup.SetBackgroundBounds(popupBounds)
	contentSize := p.contentSize(context)
	center := image.Point{
		X: popupBounds.Min.X + (popupBounds.Dx()-contentSize.X)/2,
		Y: popupBounds.Min.Y + (popupBounds.Dy()-contentSize.Y)/2,
	}
	layouter.LayoutWidget(&p.simplePopup, image.Rectangle{
		Min: center,
		Max: center.Add(contentSize),
	})
	layouter.LayoutWidget(&p.contextMenuPopup, image.Rectangle{
		Min: p.contextMenuPopupPosition,
		Max: p.contextMenuPopupPosition.Add(p.contextMenuPopup.Measure(context, guigui.Constraints{})),
	})

	u := basicwidget.UnitSize(context)
	p.layoutItems = slices.Delete(p.layoutItems, 0, len(p.layoutItems))
	p.layoutItems = append(p.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &p.forms[0],
		},
		guigui.LinearLayoutItem{
			Widget: &p.forms[1],
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     p.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

var (
	popupClickHereTextEventClicked guigui.EventKey = guigui.GenerateEventKey()
)

type popupClickHereText struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (p *popupClickHereText) Text() *basicwidget.Text {
	return &p.text
}

func (b *popupClickHereText) OnClicked(f func(context *guigui.Context, pt image.Point)) {
	guigui.SetEventHandler(b, popupClickHereTextEventClicked, f)
}

func (p *popupClickHereText) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&p.text)
	return nil
}

func (b *popupClickHereText) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&b.text, widgetBounds.Bounds())
}

func (b *popupClickHereText) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return b.text.Measure(context, constraints)
}

func (p *popupClickHereText) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if widgetBounds.IsHitAtCursor() {
			guigui.DispatchEvent(p, popupClickHereTextEventClicked, image.Pt(ebiten.CursorPosition()))
			return guigui.HandleInputByWidget(p)
		}
	}
	return guigui.HandleInputResult{}
}

type simplePopupContent struct {
	guigui.DefaultWidget

	popup *basicwidget.Popup

	titleText   basicwidget.Text
	closeButton basicwidget.Button

	buttonRowLayout guigui.LinearLayout
	buttonRowItems  []guigui.LinearLayoutItem
	layoutItems     []guigui.LinearLayoutItem
}

func (s *simplePopupContent) SetPopup(popup *basicwidget.Popup) {
	s.popup = popup
}

func (s *simplePopupContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&s.titleText)
	adder.AddWidget(&s.closeButton)
	s.titleText.SetValue("Hello!")
	s.titleText.SetBold(true)

	s.closeButton.SetText("Close")
	s.closeButton.OnUp(func(context *guigui.Context) {
		s.popup.SetOpen(false)
	})

	return nil
}

func (s *simplePopupContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
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
			Widget: &s.titleText,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(s.closeButton.Measure(context, guigui.Constraints{}).Y),
			Layout: &s.buttonRowLayout,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     s.layoutItems,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
