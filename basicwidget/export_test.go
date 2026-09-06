// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package basicwidget

import (
	"image"

	"github.com/guigui-gui/guigui"
)

type PopupState struct {
	popup
}

func (p *PopupState) Advance() error {
	var context guigui.Context
	return p.popup.Tick(&context, nil)
}

func (p *PopupState) CloseByClickingOutside() {
	var context guigui.Context
	p.close(&context, PopupCloseReasonClickOutside)
}

func (p *PopupState) Passthrough() bool {
	return p.passthrough()
}

type AbstractNumberInput = abstractNumberInput

func TopItemAfterPixelScroll(measure func(index int) int, totalCount, startIndex, startOffset, deltaPx int) (int, int) {
	return topItemAfterPixelScroll(measure, totalCount, startIndex, startOffset, deltaPx)
}

func BottomFracIdx(measure func(index int) int, totalCount, viewportHeight int) float64 {
	return bottomFracIdx(measure, totalCount, viewportHeight)
}

// VirtualScrollPanel exposes the unexported virtualScrollPanel so tests can
// exercise the scroll-position primitives that List's public scroll API
// delegates to.
type VirtualScrollPanel struct {
	virtualScrollPanel
}

func (p *VirtualScrollPanel) TopItem() (int, int) {
	return p.topItem()
}

func (p *VirtualScrollPanel) SetTopItem(index, offset int) {
	p.setTopItem(index, offset)
}

func (p *VirtualScrollPanel) ForceSetTopItem(index, offset int, cancelAnimation bool) {
	p.forceSetTopItem(index, offset, cancelAnimation)
}

func (p *VirtualScrollPanel) ForceSetScrollOffsetX(x float64) {
	p.forceSetScrollOffsetX(x)
}

func (p *VirtualScrollPanel) ApplyPendingScrollOffset() {
	p.applyPendingScrollOffset()
}

func (p *VirtualScrollPanel) ScrollOffset() (float64, float64) {
	return p.scrollOffset()
}

// TextInputText exposes the horizontal layout state of textInputText so tests
// can verify the relationship between its scroll extent and wrapping width.
type TextInputText struct {
	textInputText
}

func (t *TextInputText) ConfigureHorizontalLayout(wrapMode WrapMode, containerWidth, paddingStart, paddingEnd, measuredWidth int) {
	txt := t.text.Widget()
	txt.SetMultiline(true)
	txt.SetWrapMode(wrapMode)
	t.containerBounds = image.Rect(0, 0, containerWidth, 1)
	t.padding.Start = paddingStart
	t.padding.End = paddingEnd
	t.measuredMaxWidth = measuredWidth
	t.measuredMaxWidthWrapMode = wrapMode
	t.measuredMaxWidthInnerWidth = containerWidth - paddingStart - paddingEnd
	txt.core.SetWrapWidth(t.measuredMaxWidthInnerWidth)
}

func (t *TextInputText) ContentWidth() int {
	return t.contentWidth(nil)
}

func (t *TextInputText) LayoutWidth(bounds image.Rectangle) int {
	return t.text.Widget().core.LayoutWidth(bounds)
}

// SupportTextMinX exposes supportTextMinX so tests can verify where an
// unwrapped support text wider than the text input area extends to.
func SupportTextMinX(halign HorizontalAlign, inputBounds image.Rectangle, width int) int {
	return supportTextMinX(halign, inputBounds, width)
}

type AbstractListValuer[T comparable] interface {
	valuer[T]
}

type AbstractList[Value comparable, Item AbstractListValuer[Value]] struct {
	abstractList[Value, Item]
}

// SelectionStateKey returns the fingerprint that writeStateKey feeds into the
// StateKeyWriter, so tests can detect whether a selection change is observable
// through the state-key machinery.
func (a *AbstractList[Value, Item]) SelectionStateKey() []int {
	return a.selectionFingerprint
}

type AbstractListTestItem[T comparable] struct {
	Value      T
	Selectable bool
	Visible    bool
}

func (a AbstractListTestItem[T]) value() T {
	return a.Value
}

func (a AbstractListTestItem[T]) selectable() bool {
	return a.Selectable
}

func (a AbstractListTestItem[T]) visible() bool {
	return a.Visible
}
