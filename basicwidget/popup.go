// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

const popupZ = 16

const (
	popupEventClosed = "closed"
)

func easeOutQuad(t float64) float64 {
	// https://greweb.me/2012/02/bezier-curve-based-easing-functions-from-concept-to-implementation
	// easeOutQuad
	return t * (2 - t)
}

func popupMaxOpeningCount() int {
	return ebiten.TPS() / 5
}

type PopupClosedReason int

const (
	PopupClosedReasonNone PopupClosedReason = iota
	PopupClosedReasonFuncCall
	PopupClosedReasonClickOutside
	PopupClosedReasonReopen
)

type Popup struct {
	guigui.DefaultWidget

	blurredBackground popupBlurredBackground
	shadow            popupShadow
	content           popupContent
	frame             popupFrame

	openingCount           int
	showing                bool
	hiding                 bool
	closedReason           PopupClosedReason
	backgroundBlurred      bool
	closeByClickingOutside bool
	animateOnFading        bool
	contentPosition        image.Point
	nextContentPosition    image.Point
	hasNextContentPosition bool
	openAfterClose         bool
}

func (p *Popup) IsOpen() bool {
	return p.showing || p.hiding || p.openingCount > 0
}

func (p *Popup) SetContent(widget guigui.Widget) {
	p.content.setContent(widget)
}

func (p *Popup) openingRate() float64 {
	return easeOutQuad(float64(p.openingCount) / float64(popupMaxOpeningCount()))
}

func (p *Popup) contentBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	if p.content.content == nil {
		return image.Rectangle{}
	}
	pt := p.contentPosition
	if p.animateOnFading {
		rate := p.openingRate()
		dy := int(-float64(UnitSize(context)) * (1 - rate))
		pt = pt.Add(image.Pt(0, dy))
	}
	return image.Rectangle{
		Min: pt,
		Max: pt.Add(widgetBounds.Bounds().Size()),
	}
}

func (p *Popup) SetBackgroundBlurred(blurBackground bool) {
	p.backgroundBlurred = blurBackground
}

func (p *Popup) SetCloseByClickingOutside(closeByClickingOutside bool) {
	p.closeByClickingOutside = closeByClickingOutside
}

func (p *Popup) SetAnimationDuringFade(animateOnFading bool) {
	// TODO: Rename Popup to basePopup and create Popup with animateOnFading true.
	p.animateOnFading = animateOnFading
}

func (p *Popup) SetOnClosed(f func(reason PopupClosedReason)) {
	guigui.RegisterEventHandler(p, popupEventClosed, f)
}

func (p *Popup) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	if p.openingRate() > 0 {
		if p.backgroundBlurred {
			adder.AddChild(&p.blurredBackground)
		}
		adder.AddChild(&p.shadow)
		adder.AddChild(&p.content)
		adder.AddChild(&p.frame)
	}
}

func (p *Popup) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if (p.showing || p.hiding) && p.openingCount > 0 {
		p.nextContentPosition = widgetBounds.Bounds().Min
		p.hasNextContentPosition = true
	} else {
		p.contentPosition = widgetBounds.Bounds().Min
		p.nextContentPosition = image.Point{}
		p.hasNextContentPosition = false
	}

	p.shadow.SetContentBounds(p.contentBounds(context, widgetBounds))

	return nil
}

func (p *Popup) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case &p.blurredBackground:
		return context.AppBounds()
	case &p.shadow:
		return context.AppBounds()
	case &p.content:
		return p.contentBounds(context, widgetBounds)
	case &p.frame:
		return p.contentBounds(context, widgetBounds)
	}
	return image.Rectangle{}
}

func (p *Popup) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if p.showing || p.hiding {
		return guigui.AbortHandlingInputByWidget(p)
	}

	if !p.closeByClickingOutside {
		return guigui.AbortHandlingInputByWidget(p)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		p.close(PopupClosedReasonClickOutside)
		// Continue handling inputs so that clicking a right button can be handled by other widgets.
		// This is a little tricky, but this is needed to reopen context menu popups.
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			return guigui.HandleInputResult{}
		}
	}

	return guigui.AbortHandlingInputByWidget(p)
}

func (p *Popup) SetOpen(open bool) {
	if open {
		if p.showing {
			return
		}
		if p.openingCount > 0 {
			p.close(PopupClosedReasonReopen)
			p.openAfterClose = true
			return
		}
		p.showing = true
		p.hiding = false
		p.shadow.SetPassThrough(p.backgroundPassThrough())
	} else {
		p.close(PopupClosedReasonFuncCall)
	}
}

func (p *Popup) setClosedReason(reason PopupClosedReason) {
	if p.closedReason == PopupClosedReasonNone {
		p.closedReason = reason
		return
	}
	if reason != PopupClosedReasonReopen {
		return
	}
	// Overwrite the closed reason if it is PopupClosedReasonReopen.
	// A popup might already be closed by clicking outside.
	p.closedReason = reason
}

func (p *Popup) close(reason PopupClosedReason) {
	if p.hiding {
		p.setClosedReason(reason)
		return
	}
	if p.openingCount == 0 {
		return
	}

	p.setClosedReason(reason)
	p.showing = false
	p.hiding = true
	p.openAfterClose = false
	p.shadow.SetPassThrough(p.backgroundPassThrough())
}

func (p *Popup) backgroundPassThrough() bool {
	return p.openingCount == 0 || p.showing || p.hiding
}

func (p *Popup) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if p.showing {
		context.SetFocused(p, true)
		if p.openingCount < popupMaxOpeningCount() {
			p.openingCount += 3
			p.openingCount = min(p.openingCount, popupMaxOpeningCount())
		}
		if p.openingCount == popupMaxOpeningCount() {
			p.showing = false
			if p.hasNextContentPosition {
				p.contentPosition = p.nextContentPosition
				p.hasNextContentPosition = false
			}
		}
	}
	if p.hiding {
		if 0 < p.openingCount {
			if p.closedReason == PopupClosedReasonReopen {
				p.openingCount -= 3
			} else {
				p.openingCount--
			}
			p.openingCount = max(p.openingCount, 0)
		}
		if p.openingCount == 0 {
			context.SetFocused(p, false)
			p.hiding = false
			guigui.DispatchEventHandler(p, popupEventClosed, p.closedReason)
			p.closedReason = PopupClosedReasonNone
			if p.openAfterClose {
				if p.hasNextContentPosition {
					p.contentPosition = p.nextContentPosition
					p.hasNextContentPosition = false
				}
				p.SetOpen(true)
				p.openAfterClose = false
			}
		}
	}

	p.shadow.SetPassThrough(p.backgroundPassThrough())
	p.blurredBackground.SetOpeningRate(p.openingRate())

	// SetOpacity cannot be called for p.blurredBackground so far.
	// If opacity is less than 1, the dst argument of Draw will an empty image in the current implementation.
	// TODO: This is too tricky. Refactor this.
	context.SetOpacity(&p.shadow, p.openingRate())
	context.SetOpacity(&p.content, p.openingRate())
	context.SetOpacity(&p.frame, p.openingRate())

	return nil
}

func (p *Popup) ZDelta() int {
	return popupZ
}

func (p *Popup) PassThrough() bool {
	return !p.IsOpen()
}

type popupContent struct {
	guigui.DefaultWidget

	content guigui.Widget
}

func (p *popupContent) setContent(widget guigui.Widget) {
	p.content = widget
}

func (p *popupContent) AddChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, adder *guigui.ChildAdder) {
	if p.content != nil {
		adder.AddChild(p.content)
	}
}

func (p *popupContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	switch widget {
	case p.content:
		return widgetBounds.Bounds()
	}
	return image.Rectangle{}
}

func (p *popupContent) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if context.IsWidgetHitAtCursor(p) {
		return guigui.AbortHandlingInputByWidget(p)
	}
	return guigui.HandleInputResult{}
}

func (p *popupContent) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := draw.Color(context.ColorMode(), draw.ColorTypeBase, 1)
	draw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
}

func (p *popupContent) ZDelta() int {
	return 1
}

type popupFrame struct {
	guigui.DefaultWidget
}

func (p *popupFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr1, clr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderTypeOutset, false)
	draw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), float32(1*context.Scale()), draw.RoundedRectBorderTypeOutset)
}

func (p *popupFrame) ZDelta() int {
	return 1
}

type popupBlurredBackground struct {
	guigui.DefaultWidget

	backgroundCache *ebiten.Image

	openingRate float64
}

func (p *popupBlurredBackground) SetOpeningRate(rate float64) {
	if p.openingRate == rate {
		return
	}
	p.openingRate = rate
	guigui.RequestRedraw(p)
}

func (p *popupBlurredBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	if p.backgroundCache != nil && !bounds.In(p.backgroundCache.Bounds()) {
		p.backgroundCache.Deallocate()
		p.backgroundCache = nil
	}
	if p.backgroundCache == nil {
		p.backgroundCache = ebiten.NewImageWithOptions(bounds, nil)
	}

	rate := p.openingRate

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(dst.Bounds().Min.X), float64(dst.Bounds().Min.Y))
	op.Blend = ebiten.BlendCopy
	p.backgroundCache.DrawImage(dst, op)

	draw.DrawBlurredImage(context, dst, p.backgroundCache, rate)
}

func (p *popupBlurredBackground) ZDelta() int {
	return 1
}

type popupShadow struct {
	guigui.DefaultWidget

	contentBounds image.Rectangle

	passThrough bool
}

func (p *popupShadow) SetContentBounds(bounds image.Rectangle) {
	if p.contentBounds == bounds {
		return
	}
	p.contentBounds = bounds
	guigui.RequestRedraw(p)
}

func (p *popupShadow) SetPassThrough(passThrough bool) {
	p.passThrough = passThrough
}

func (p *popupShadow) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := p.contentBounds
	bounds.Min.X -= int(16 * context.Scale())
	bounds.Max.X += int(16 * context.Scale())
	bounds.Min.Y -= int(8 * context.Scale())
	bounds.Max.Y += int(16 * context.Scale())
	clr := draw.ScaleAlpha(color.Black, 0.2)
	draw.DrawRoundedShadowRect(context, dst, bounds, clr, int(16*context.Scale())+RoundedCornerRadius(context))
}

func (p *popupShadow) ZDelta() int {
	return 1
}

func (p *popupShadow) PassThrough() bool {
	return p.passThrough
}
