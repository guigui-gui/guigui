// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"

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
	showButton                   basicwidget.Button

	contextMenuPopupText          basicwidget.Text
	contextMenuPopupClickHereText popupClickHereText

	simplePopup        basicwidget.Popup
	simplePopupContent guigui.WidgetWithSize[*simplePopupContent]

	contextMenuPopup basicwidget.PopupMenu[int]

	contextMenuPopupPosition image.Point
}

func (p *Popups) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	for i := range p.forms {
		adder.AddChild(&p.forms[i])
	}
	adder.AddChild(&p.simplePopup)
	adder.AddChild(&p.contextMenuPopup)
}

func (p *Popups) Update(context *guigui.Context) error {
	p.darkenBackgroundText.SetValue("Darken background")
	p.blurBackgroundText.SetValue("Blur background")
	p.closeByClickingOutsideText.SetValue("Close by clicking outside")
	p.showButton.SetText("Show")
	p.showButton.SetOnUp(func() {
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
	p.simplePopup.SetBackgroundDarkened(p.darkenBackgroundToggle.Value())
	p.simplePopup.SetBackgroundBlurred(p.blurBackgroundToggle.Value())
	p.simplePopup.SetCloseByClickingOutside(p.closeByClickingOutsideToggle.Value())
	p.simplePopup.SetAnimationDuringFade(true)

	p.simplePopupContent.SetFixedSize(p.contentSize(context))

	p.contextMenuPopup.SetItems(
		[]basicwidget.PopupMenuItem[int]{
			{
				Text: "Item 1",
			},
			{
				Text: "Item 2",
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

	p.contextMenuPopupClickHereText.SetOnClicked(func(pt image.Point) {
		p.contextMenuPopupPosition = pt
		p.contextMenuPopup.SetOpen(true)
	})

	return nil
}

func (p *Popups) contentSize(context *guigui.Context) image.Point {
	u := basicwidget.UnitSize(context)
	return image.Pt(int(12*u), int(6*u))
}

func (p *Popups) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	appBounds := context.AppBounds()
	contentSize := p.contentSize(context)
	center := image.Point{
		X: appBounds.Min.X + (appBounds.Dx()-contentSize.X)/2,
		Y: appBounds.Min.Y + (appBounds.Dy()-contentSize.Y)/2,
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
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &p.forms[0],
			},
			{
				Widget: &p.forms[1],
			},
		},
		Gap: u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

const popupClickHereTextEventClicked = "clicked"

type popupClickHereText struct {
	guigui.DefaultWidget

	text basicwidget.Text
}

func (p *popupClickHereText) Text() *basicwidget.Text {
	return &p.text
}

func (b *popupClickHereText) SetOnClicked(f func(pt image.Point)) {
	guigui.RegisterEventHandler(b, popupClickHereTextEventClicked, f)
}

func (p *popupClickHereText) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&p.text)
}

func (b *popupClickHereText) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&b.text, widgetBounds.Bounds())
}

func (b *popupClickHereText) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return b.text.Measure(context, constraints)
}

func (p *popupClickHereText) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if widgetBounds.IsHitAtCursor() {
			guigui.DispatchEventHandler(p, popupClickHereTextEventClicked, image.Pt(ebiten.CursorPosition()))
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
}

func (s *simplePopupContent) SetPopup(popup *basicwidget.Popup) {
	s.popup = popup
}

func (s *simplePopupContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&s.titleText)
	adder.AddChild(&s.closeButton)
}

func (s *simplePopupContent) Update(context *guigui.Context) error {
	s.titleText.SetValue("Hello!")
	s.titleText.SetBold(true)

	s.closeButton.SetText("Close")
	s.closeButton.SetOnUp(func() {
		s.popup.SetOpen(false)
	})

	return nil
}

func (s *simplePopupContent) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &s.titleText,
				Size:   guigui.FlexibleSize(1),
			},
			{
				Size: guigui.FixedSize(s.closeButton.Measure(context, guigui.Constraints{}).Y),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionHorizontal,
					Items: []guigui.LinearLayoutItem{
						{
							Size: guigui.FlexibleSize(1),
						},
						{
							Widget: &s.closeButton,
						},
					},
				},
			},
		},
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
