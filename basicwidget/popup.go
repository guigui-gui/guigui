// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget/basicwidgetdraw"
	"github.com/guigui-gui/guigui/basicwidget/internal/draw"
)

const popupZ = 16

const (
	popupEventClose = "close"
)

func easeOutQuad(t float64) float64 {
	// https://greweb.me/2012/02/bezier-curve-based-easing-functions-from-concept-to-implementation
	// easeOutQuad
	return t * (2 - t)
}

func popupMaxOpeningCount() int {
	return ebiten.TPS() / 5
}

type popupStyle int

const (
	popupStyleNormal popupStyle = iota
	popupStyleMenu
	popupStyleDrawer
)

type PopupCloseReason int

const (
	PopupCloseReasonNone PopupCloseReason = iota
	PopupCloseReasonFuncCall
	PopupCloseReasonClickOutside
	PopupCloseReasonReopen
)

type Popup struct {
	guigui.DefaultWidget

	popup popup
}

func (p *Popup) setStyle(style popupStyle) {
	p.popup.setStyle(style)
}

func (p *Popup) SetOpen(open bool) {
	p.popup.SetOpen(open)
}

func (p *Popup) IsOpen() bool {
	return p.popup.IsOpen()
}

func (p *Popup) SetOnClose(f func(context *guigui.Context, reason PopupCloseReason)) {
	p.popup.SetOnClose(f)
}

func (p *Popup) SetContent(widget guigui.Widget) {
	p.popup.SetContent(widget)
}

func (p *Popup) SetBackgroundDark(dark bool) {
	p.popup.SetBackgroundDark(dark)
}

func (p *Popup) SetBackgroundBlurred(blurred bool) {
	p.popup.SetBackgroundBlurred(blurred)
}

func (p *Popup) SetCloseByClickingOutside(closeByClickingOutside bool) {
	p.popup.SetCloseByClickingOutside(closeByClickingOutside)
}

func (p *Popup) SetAnimationDuringFade(animateOnFading bool) {
	p.popup.SetAnimationDuringFade(animateOnFading)
}

func (p *Popup) SetBackgroundBounds(bounds image.Rectangle) {
	p.popup.SetBackgroundBounds(bounds)
}

func (p *Popup) setDrawerEdge(edge DrawerEdge) {
	p.popup.SetDrawerEdge(edge)
}

func (p *Popup) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&p.popup)
	context.SetPassThrough(&p.popup, !p.IsOpen())
	context.SetZDelta(&p.popup, popupZ)
	context.SetContainer(&p.popup, true)
	return nil
}

func (p *Popup) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&p.popup, widgetBounds.Bounds())
}

func (p *Popup) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return p.popup.Measure(context, constraints)
}

type popup struct {
	guigui.DefaultWidget

	blurredBackground popupBlurredBackground
	darkBackground    popupDarkBackground
	shadow            popupShadow
	contentAndFrame   popupContentAndFrame

	style                  popupStyle
	toOpen                 bool
	toClose                bool
	openingCount           int
	showing                bool
	hiding                 bool
	closeReason            PopupCloseReason
	backgroundDark         bool
	backgroundBlurred      bool
	closeByClickingOutside bool
	animateOnFading        bool
	contentPosition        image.Point
	nextContentPosition    image.Point
	hasNextContentPosition bool
	openAfterClose         bool
	backgroundBounds       image.Rectangle
	drawerEdge             DrawerEdge
}

func (p *popup) setStyle(style popupStyle) {
	if p.style == style {
		return
	}
	p.style = style
	p.contentAndFrame.setStyle(style)
	p.shadow.setStyle(style)
	guigui.RequestRedraw(p)
}

func (p *popup) IsOpen() bool {
	return p.showing || p.hiding || p.openingCount > 0 || p.toOpen
}

func (p *popup) SetContent(widget guigui.Widget) {
	p.contentAndFrame.setContent(widget)
}

func (p *popup) openingRate() float64 {
	return easeOutQuad(float64(p.openingCount) / float64(popupMaxOpeningCount()))
}

func (p *popup) contentBounds(context *guigui.Context, widgetBounds *guigui.WidgetBounds) image.Rectangle {
	pt := p.contentPosition
	if p.animateOnFading {
		rate := p.openingRate()
		if p.style != popupStyleDrawer {
			dy := int(-float64(UnitSize(context)) * (1 - rate))
			pt = pt.Add(image.Pt(0, dy))
		} else {
			bgBounds := p.backgroundBounds
			if bgBounds.Empty() {
				bgBounds = context.AppBounds()
			}
			srcPt := widgetBounds.Bounds().Min
			switch p.drawerEdge {
			case DrawerEdgeStart:
				srcPt.X = bgBounds.Min.X - widgetBounds.Bounds().Dx()
			case DrawerEdgeTop:
				srcPt.Y = bgBounds.Min.Y - widgetBounds.Bounds().Dy()
			case DrawerEdgeEnd:
				srcPt.X = bgBounds.Max.X
			case DrawerEdgeBottom:
				srcPt.Y = bgBounds.Max.Y
			}
			dstPt := p.contentPosition
			// Mix srcPt and dstPt by rate.
			pt = image.Pt(
				int(float64(srcPt.X)*(1-rate)+float64(dstPt.X)*rate),
				int(float64(srcPt.Y)*(1-rate)+float64(dstPt.Y)*rate),
			)
		}
	}
	return image.Rectangle{
		Min: pt,
		Max: pt.Add(widgetBounds.Bounds().Size()),
	}
}

func (p *popup) SetBackgroundDark(dark bool) {
	p.backgroundDark = dark
}

func (p *popup) SetBackgroundBlurred(blurred bool) {
	p.backgroundBlurred = blurred
}

func (p *popup) SetCloseByClickingOutside(closeByClickingOutside bool) {
	p.closeByClickingOutside = closeByClickingOutside
}

func (p *popup) SetAnimationDuringFade(animateOnFading bool) {
	// TODO: Rename Popup to basePopup and create Popup with animateOnFading true.
	p.animateOnFading = animateOnFading
}

func (p *popup) SetBackgroundBounds(bounds image.Rectangle) {
	p.backgroundBounds = bounds
}

func (p *popup) SetDrawerEdge(edge DrawerEdge) {
	p.drawerEdge = edge
	p.contentAndFrame.setDrawerEdge(edge)
}

func (p *popup) SetOnClose(f func(context *guigui.Context, reason PopupCloseReason)) {
	guigui.SetEventHandler(p, popupEventClose, f)
}

func (p *popup) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if p.openingRate() > 0 {
		if p.backgroundBlurred {
			adder.AddChild(&p.blurredBackground)
		}
		if p.backgroundDark {
			adder.AddChild(&p.darkBackground)
		}
		adder.AddChild(&p.shadow)
		adder.AddChild(&p.contentAndFrame)
	}

	context.SetZDelta(&p.blurredBackground, 1)
	context.SetZDelta(&p.darkBackground, 1)
	context.SetZDelta(&p.shadow, 1)
	context.SetZDelta(&p.contentAndFrame, 1)
	return nil
}

func (p *popup) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	if (p.hiding || p.toClose) && p.openingCount > 0 {
		// When the popup is fading out, keep the current position.
		// This matters especially when the same popup menu is reopened at a different position.
		// p.showing is ignored here because the position might be updated soon after opening.
		p.nextContentPosition = widgetBounds.Bounds().Min
		p.hasNextContentPosition = true
	} else {
		p.contentPosition = widgetBounds.Bounds().Min
		p.nextContentPosition = image.Point{}
		p.hasNextContentPosition = false
	}
	contentBounds := p.contentBounds(context, widgetBounds)
	p.shadow.SetContentBounds(contentBounds)

	bounds := p.backgroundBounds
	if bounds.Empty() {
		bounds = context.AppBounds()
	}
	layouter.LayoutWidget(&p.blurredBackground, bounds)
	layouter.LayoutWidget(&p.darkBackground, bounds)
	layouter.LayoutWidget(&p.shadow, bounds)
	layouter.LayoutWidget(&p.contentAndFrame, contentBounds)
}

func (p *popup) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if p.openingRate() == 0 {
		return guigui.HandleInputResult{}
	}

	bounds := p.backgroundBounds
	if bounds.Empty() {
		bounds = context.AppBounds()
	}
	if !image.Pt(ebiten.CursorPosition()).In(bounds) {
		return guigui.HandleInputResult{}
	}

	if p.showing || p.hiding {
		return guigui.AbortHandlingInputByWidget(p)
	}

	if !p.closeByClickingOutside {
		return guigui.AbortHandlingInputByWidget(p)
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		p.close(context, PopupCloseReasonClickOutside)
		// Continue handling inputs so that clicking a right button can be handled by other widgets.
		// This is a little tricky, but this is needed to reopen context menu popups.
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			return guigui.HandleInputResult{}
		}
	}

	return guigui.AbortHandlingInputByWidget(p)
}

func (p *popup) SetOpen(open bool) {
	if open {
		p.toOpen = true
		p.toClose = false
	} else {
		p.toOpen = false
		p.toClose = true
	}
}

func (p *popup) setCloseReason(reason PopupCloseReason) {
	if p.closeReason == PopupCloseReasonNone {
		p.closeReason = reason
		return
	}
	if reason != PopupCloseReasonReopen {
		return
	}
	// Overwrite the closed reason if it is PopupCloseReasonReopen.
	// A popup might already be closed by clicking outside.
	p.closeReason = reason
}

func (p *popup) close(context *guigui.Context, reason PopupCloseReason) {
	if p.hiding {
		p.setCloseReason(reason)
		return
	}
	if p.openingCount == 0 {
		return
	}

	p.setCloseReason(reason)
	p.showing = false
	p.hiding = true
	p.openAfterClose = false
	// TODO: Is this needed?
	context.SetPassThrough(&p.shadow, p.backgroundPassThrough())
}

func (p *popup) backgroundPassThrough() bool {
	return p.openingCount == 0 || p.showing || p.hiding
}

func (p *popup) canUpdateContent() bool {
	if p.hiding {
		return false
	}
	return true
}

func (p *popup) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	if p.toOpen {
		if !p.showing {
			if p.openingCount > 0 {
				p.close(context, PopupCloseReasonReopen)
				p.openAfterClose = true
			} else {
				p.showing = true
				p.hiding = false
			}
		}
	} else if p.toClose {
		p.close(context, PopupCloseReasonFuncCall)
	}
	p.toOpen = false
	p.toClose = false

	if p.showing {
		if p.openingCount < popupMaxOpeningCount() {
			if p.style == popupStyleMenu {
				p.openingCount += 4
			} else {
				p.openingCount += 2
			}
			p.openingCount = min(p.openingCount, popupMaxOpeningCount())
			guigui.RequestRedraw(p)
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
			if p.closeReason == PopupCloseReasonReopen || p.style == popupStyleMenu {
				p.openingCount -= 4
			} else {
				p.openingCount--
			}
			p.openingCount = max(p.openingCount, 0)
			guigui.RequestRedraw(p)
		}
		if p.openingCount == 0 {
			p.hiding = false
			guigui.DispatchEvent(p, popupEventClose, p.closeReason)
			p.closeReason = PopupCloseReasonNone
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

	// TODO: Is this needed?
	context.SetPassThrough(&p.shadow, p.backgroundPassThrough())
	p.blurredBackground.SetOpeningRate(p.openingRate())
	p.darkBackground.SetOpeningRate(p.openingRate())
	p.shadow.SetOpeningRate(p.openingRate())

	var opacity float64
	if p.style != popupStyleDrawer {
		opacity = p.openingRate()
	} else {
		opacity = 1
	}
	// SetOpacity cannot be called for p.blurredBackground so far.
	// If opacity is less than 1, the dst argument of Draw will an empty image in the current implementation.
	// Use an original implementation by SetOpeningRate anyway as this is more performant.
	// TODO: This is too tricky. Refactor this.
	context.SetOpacity(&p.contentAndFrame, opacity)

	return nil
}

type popupContentAndFrame struct {
	guigui.DefaultWidget

	content popupContent
	frame   popupFrame
	style   popupStyle
}

func (p *popupContentAndFrame) setContent(widget guigui.Widget) {
	p.content.setContent(widget)
}

func (p *popupContentAndFrame) setStyle(style popupStyle) {
	p.style = style
	p.content.setStyle(style)
	p.frame.setStyle(style)
}

func (p *popupContentAndFrame) setDrawerEdge(edge DrawerEdge) {
	p.frame.setDrawerEdge(edge)
}

func (p *popupContentAndFrame) hasContent() bool {
	return p.content.hasContent()
}

func (p *popupContentAndFrame) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&p.content)
	adder.AddChild(&p.frame)
	return nil
}

func (p *popupContentAndFrame) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	if p.style != popupStyleDrawer {
		// CustomDraw might be too generic and overkill for this case.
		context.SetCustomDraw(p, func(dst, widgetImage *ebiten.Image, op *ebiten.DrawImageOptions) {
			draw.DrawInRoundedCornerRect(context, dst, widgetBounds.Bounds(), RoundedCornerRadius(context), widgetImage, op)
		})
	}
	layouter.LayoutWidget(&p.content, widgetBounds.Bounds())
	layouter.LayoutWidget(&p.frame, widgetBounds.Bounds())
}

type popupContent struct {
	guigui.DefaultWidget

	content guigui.Widget
	style   popupStyle
}

func (p *popupContent) setContent(widget guigui.Widget) {
	p.content = widget
}

func (p *popupContent) setStyle(style popupStyle) {
	p.style = style
}

func (p *popupContent) hasContent() bool {
	return p.content != nil
}

func (p *popupContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if p.content != nil {
		adder.AddChild(p.content)
	}
	return nil
}

func (p *popupContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	if p.content != nil {
		layouter.LayoutWidget(p.content, widgetBounds.Bounds())
	}
}

func (p *popupContent) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() {
		return guigui.AbortHandlingInputByWidget(p)
	}
	return guigui.HandleInputResult{}
}

func (p *popupContent) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr := draw.Color(context.ColorMode(), draw.ColorTypeBase, 1)
	if p.style != popupStyleDrawer {
		basicwidgetdraw.DrawRoundedRect(context, dst, bounds, clr, RoundedCornerRadius(context))
	} else {
		vector.FillRect(dst, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()), clr, false)
	}
}

type popupFrame struct {
	guigui.DefaultWidget

	style      popupStyle
	drawerEdge DrawerEdge
}

func (p *popupFrame) setStyle(style popupStyle) {
	p.style = style
}

func (p *popupFrame) setDrawerEdge(edge DrawerEdge) {
	p.drawerEdge = edge
}

func (p *popupFrame) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()
	clr1, clr2 := basicwidgetdraw.BorderColors(context.ColorMode(), basicwidgetdraw.RoundedRectBorderTypeOutset, false)

	width := float32(1 * context.Scale())
	if p.style != popupStyleDrawer {
		basicwidgetdraw.DrawRoundedRectBorder(context, dst, bounds, clr1, clr2, RoundedCornerRadius(context), width, basicwidgetdraw.RoundedRectBorderTypeOutset)
	} else {
		var x0, y0, x1, y1 float32
		switch p.drawerEdge {
		case DrawerEdgeStart:
			x0 = float32(bounds.Max.X) - width/2
			y0 = float32(bounds.Min.Y)
			x1 = float32(bounds.Max.X) - width/2
			y1 = float32(bounds.Max.Y)
		case DrawerEdgeTop:
			x0 = float32(bounds.Min.X)
			y0 = float32(bounds.Max.Y) - width/2
			x1 = float32(bounds.Max.X)
			y1 = float32(bounds.Max.Y) - width/2
		case DrawerEdgeEnd:
			x0 = float32(bounds.Min.X) + width/2
			y0 = float32(bounds.Min.Y)
			x1 = float32(bounds.Min.X) + width/2
			y1 = float32(bounds.Max.Y)
		case DrawerEdgeBottom:
			x0 = float32(bounds.Min.X)
			y0 = float32(bounds.Min.Y) + width/2
			x1 = float32(bounds.Max.X)
			y1 = float32(bounds.Min.Y) + width/2
		}
		// TODO: Use clr2 as a gradation
		vector.StrokeLine(dst, x0, y0, x1, y1, width, clr1, true)
	}
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

type popupDarkBackground struct {
	guigui.DefaultWidget

	openingRate float64
}

func (p *popupDarkBackground) SetOpeningRate(rate float64) {
	if p.openingRate == rate {
		return
	}
	p.openingRate = rate
	guigui.RequestRedraw(p)
}

func (p *popupDarkBackground) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := widgetBounds.Bounds()

	clr := draw.ScaleAlpha(draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0.7, 0.1), 0.75*p.openingRate)
	vector.FillRect(dst, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()), clr, false)
}

type popupShadow struct {
	guigui.DefaultWidget

	contentBounds image.Rectangle
	openingRate   float64
	style         popupStyle
}

func (p *popupShadow) SetOpeningRate(rate float64) {
	if p.openingRate == rate {
		return
	}
	p.openingRate = rate
	guigui.RequestRedraw(p)
}

func (p *popupShadow) SetContentBounds(bounds image.Rectangle) {
	if p.contentBounds == bounds {
		return
	}
	p.contentBounds = bounds
	guigui.RequestRedraw(p)
}

func (p *popupShadow) setStyle(style popupStyle) {
	p.style = style
}

func (p *popupShadow) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	bounds := p.contentBounds
	bounds.Min.X -= int(16 * context.Scale())
	bounds.Max.X += int(16 * context.Scale())
	bounds.Min.Y -= int(8 * context.Scale())
	bounds.Max.Y += int(16 * context.Scale())
	// TODO: When openingRate < 1, only the edges should be rendered.
	clr := draw.Color2(context.ColorMode(), draw.ColorTypeBase, 0, 0)
	alpha := 0.25
	if p.style != popupStyleDrawer {
		alpha *= p.openingRate
	}
	clr = draw.ScaleAlpha(clr, alpha)
	draw.DrawRoundedShadowRect(context, dst, bounds, clr, int(16*context.Scale())+RoundedCornerRadius(context))
}
