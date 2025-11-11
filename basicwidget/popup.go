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

	popup popup
}

func (p *Popup) SetOpen(context *guigui.Context, open bool) {
	p.popup.SetOpen(context, open)
}

func (p *Popup) IsOpen() bool {
	return p.popup.IsOpen()
}

func (p *Popup) SetOnClosed(f func(reason PopupClosedReason)) {
	p.popup.SetOnClosed(f)
}

func (p *Popup) SetContent(widget guigui.Widget) {
	p.popup.SetContent(widget)
}

func (p *Popup) SetBackgroundBlurred(blurBackground bool) {
	p.popup.SetBackgroundBlurred(blurBackground)
}

func (p *Popup) SetCloseByClickingOutside(closeByClickingOutside bool) {
	p.popup.SetCloseByClickingOutside(closeByClickingOutside)
}

func (p *Popup) SetAnimationDuringFade(animateOnFading bool) {
	p.popup.SetAnimationDuringFade(animateOnFading)
}

func (p *Popup) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&p.popup)
}

func (p *Popup) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	context.SetPassThrough(&p.popup, !p.IsOpen())
	context.SetZDelta(&p.popup, popupZ)
	return nil
}

func (p *Popup) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&p.popup, widgetBounds.Bounds())
}

func (p *Popup) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return p.popup.Measure(context, constraints)
}

type popup struct {
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

func (p *popup) IsOpen() bool {
	return p.showing || p.hiding || p.openingCount > 0
}

func (p *popup) SetContent(widget guigui.Widget) {
	p.content.setContent(widget)
}

func (p *popup) openingRate() float64 {
	return easeOutQuad(float64(p.openingCount) / float64(popupMaxOpeningCount()))
}

func (p *popup) contentBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
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

func (p *popup) SetBackgroundBlurred(blurBackground bool) {
	p.backgroundBlurred = blurBackground
}

func (p *popup) SetCloseByClickingOutside(closeByClickingOutside bool) {
	p.closeByClickingOutside = closeByClickingOutside
}

func (p *popup) SetAnimationDuringFade(animateOnFading bool) {
	// TODO: Rename Popup to basePopup and create Popup with animateOnFading true.
	p.animateOnFading = animateOnFading
}

func (p *popup) SetOnClosed(f func(reason PopupClosedReason)) {
	guigui.RegisterEventHandler(p, popupEventClosed, f)
}

func (p *popup) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if p.openingRate() > 0 {
		if p.backgroundBlurred {
			adder.AddChild(&p.blurredBackground)
		}
		adder.AddChild(&p.shadow)
		adder.AddChild(&p.content)
		adder.AddChild(&p.frame)
	}
}

func (p *popup) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
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

func (p *popup) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	contentBounds := p.contentBounds(context, widgetBounds)
	appBounds := context.AppBounds()
	layouter.LayoutWidget(&p.blurredBackground, appBounds)
	layouter.LayoutWidget(&p.shadow, appBounds)
	layouter.LayoutWidget(&p.content, contentBounds)
	layouter.LayoutWidget(&p.frame, contentBounds)
}

func (p *popup) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if p.showing || p.hiding {
		return guigui.AbortHandlingInputByWidget(p)
	}

	if !p.closeByClickingOutside {
		return guigui.AbortHandlingInputByWidget(p)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		p.close(context, PopupClosedReasonClickOutside)
		// Continue handling inputs so that clicking a right button can be handled by other widgets.
		// This is a little tricky, but this is needed to reopen context menu popups.
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			return guigui.HandleInputResult{}
		}
	}

	return guigui.AbortHandlingInputByWidget(p)
}

func (p *popup) SetOpen(context *guigui.Context, open bool) {
	if open {
		if p.showing {
			return
		}
		if p.openingCount > 0 {
			p.close(context, PopupClosedReasonReopen)
			p.openAfterClose = true
			return
		}
		p.showing = true
		p.hiding = false
	} else {
		p.close(context, PopupClosedReasonFuncCall)
	}
}

func (p *popup) setClosedReason(reason PopupClosedReason) {
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

func (p *popup) close(context *guigui.Context, reason PopupClosedReason) {
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
	context.SetPassThrough(&p.shadow, p.backgroundPassThrough())
}

func (p *popup) backgroundPassThrough() bool {
	return p.openingCount == 0 || p.showing || p.hiding
}

func (p *popup) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if p.showing {
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
			p.hiding = false
			guigui.DispatchEventHandler(p, popupEventClosed, p.closedReason)
			p.closedReason = PopupClosedReasonNone
			if p.openAfterClose {
				if p.hasNextContentPosition {
					p.contentPosition = p.nextContentPosition
					p.hasNextContentPosition = false
				}
				p.SetOpen(context, true)
				p.openAfterClose = false
			}
		}
	}

	context.SetPassThrough(&p.shadow, p.backgroundPassThrough())
	p.blurredBackground.SetOpeningRate(p.openingRate())

	// SetOpacity cannot be called for p.blurredBackground so far.
	// If opacity is less than 1, the dst argument of Draw will an empty image in the current implementation.
	// TODO: This is too tricky. Refactor this.
	context.SetOpacity(&p.shadow, p.openingRate())
	context.SetOpacity(&p.content, p.openingRate())
	context.SetOpacity(&p.frame, p.openingRate())

	context.SetZDelta(&p.blurredBackground, 1)
	context.SetZDelta(&p.shadow, 1)
	context.SetZDelta(&p.content, 1)
	context.SetZDelta(&p.frame, 1)

	return nil
}

type popupContent struct {
	guigui.DefaultWidget

	content guigui.Widget
}

func (p *popupContent) setContent(widget guigui.Widget) {
	p.content = widget
}

func (p *popupContent) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	if p.content != nil {
		adder.AddChild(p.content)
	}
}

func (p *popupContent) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	// CustomDraw might be too generic and overkill for this case.
	context.SetCustomDraw(p.content, func(dst, widgetImage *ebiten.Image, op *ebiten.DrawImageOptions) {
		draw.DrawInRoundedCornerRect(context, dst, widgetBounds.Bounds(), RoundedCornerRadius(context), widgetImage, op)
	})
	return nil
}

func (p *popupContent) LayoutChildren(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	if p.content != nil {
		layouter.LayoutWidget(p.content, widgetBounds.Bounds())
	}
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

type popupFrame struct {
	guigui.DefaultWidget
}

func (p *popupFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr1, clr2 := draw.BorderColors(context.ColorMode(), draw.RoundedRectBorderTypeOutset, false)
	draw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), float32(1*context.Scale()), draw.RoundedRectBorderTypeOutset)
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

type popupShadow struct {
	guigui.DefaultWidget

	contentBounds image.Rectangle
}

func (p *popupShadow) SetContentBounds(bounds image.Rectangle) {
	if p.contentBounds == bounds {
		return
	}
	p.contentBounds = bounds
	guigui.RequestRedraw(p)
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
