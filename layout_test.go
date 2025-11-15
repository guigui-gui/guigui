// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui_test

import (
	"image"
	"testing"

	"github.com/guigui-gui/guigui"
)

type dummyWidget struct {
	guigui.DefaultWidget

	size     image.Point
	sizeFunc func(constraints guigui.Constraints) image.Point
}

func (d *dummyWidget) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	if d.sizeFunc != nil {
		return d.sizeFunc(constraints)
	}
	return d.size
}

func TestLinearLayoutMeasure(t *testing.T) {
	l := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &dummyWidget{
					size: image.Pt(100, 200),
				},
			},
		},
	}
	var context guigui.Context
	if got, want := l.Measure(&context, guigui.Constraints{}), image.Pt(100, 200); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	for _, dir := range []guigui.LayoutDirection{guigui.LayoutDirectionHorizontal, guigui.LayoutDirectionVertical} {
		l2 := guigui.LinearLayout{
			Direction: dir,
			Items: []guigui.LinearLayoutItem{
				{
					Layout: l,
				},
			},
		}
		if got, want := l2.Measure(&context, guigui.Constraints{}), image.Pt(100, 200); got != want {
			t.Errorf("dir: %v, got: %v, want: %v", dir, got, want)
		}
	}
}

func TestLinearLayoutMeasureFlexibleSize(t *testing.T) {
	for _, dir := range []guigui.LayoutDirection{guigui.LayoutDirectionHorizontal, guigui.LayoutDirectionVertical} {
		l := guigui.LinearLayout{
			Direction: dir,
			Gap:       10,
			Items: []guigui.LinearLayoutItem{
				{
					Widget: &dummyWidget{
						sizeFunc: func(constraints guigui.Constraints) image.Point {
							if fixedWidth, ok := constraints.FixedWidth(); ok {
								return image.Pt(fixedWidth, fixedWidth)
							}
							if fixedHeight, ok := constraints.FixedHeight(); ok {
								return image.Pt(fixedHeight, fixedHeight)
							}
							return image.Pt(10, 10)
						},
					},
					Size: guigui.FlexibleSize(1),
				},
				{
					Widget: &dummyWidget{
						sizeFunc: func(constraints guigui.Constraints) image.Point {
							if fixedWidth, ok := constraints.FixedWidth(); ok {
								return image.Pt(fixedWidth, fixedWidth)
							}
							if fixedHeight, ok := constraints.FixedHeight(); ok {
								return image.Pt(fixedHeight, fixedHeight)
							}
							return image.Pt(10, 10)
						},
					},
					Size: guigui.FlexibleSize(1),
				},
			},
		}
		var context guigui.Context
		var constraints guigui.Constraints
		switch dir {
		case guigui.LayoutDirectionHorizontal:
			constraints = guigui.FixedWidthConstraints(210)
		case guigui.LayoutDirectionVertical:
			constraints = guigui.FixedHeightConstraints(210)
		}
		got := l.Measure(&context, constraints)
		var want image.Point
		switch dir {
		case guigui.LayoutDirectionHorizontal:
			want = image.Pt(210, 100)
		case guigui.LayoutDirectionVertical:
			want = image.Pt(100, 210)
		}
		if got != want {
			t.Errorf("dir: %v, got: %v, want: %v", dir, got, want)
		}
	}
}

func TestLinearLayoutMeasureIgnoreFlexibleSize(t *testing.T) {
	for _, dir := range []guigui.LayoutDirection{guigui.LayoutDirectionHorizontal, guigui.LayoutDirectionVertical} {
		var opDir guigui.LayoutDirection
		if dir == guigui.LayoutDirectionHorizontal {
			opDir = guigui.LayoutDirectionVertical
		} else {
			opDir = guigui.LayoutDirectionHorizontal
		}
		l := guigui.LinearLayout{
			Direction: dir,
			Gap:       10,
			Items: []guigui.LinearLayoutItem{
				{
					Layout: guigui.LinearLayout{
						Direction: opDir,
						Items: []guigui.LinearLayoutItem{
							{
								Size: guigui.FlexibleSize(1),
							},
							{
								Widget: &dummyWidget{
									size: image.Pt(100, 50),
								},
							},
						},
					},
				},
			},
		}
		var context guigui.Context
		got := l.Measure(&context, guigui.Constraints{})
		var want image.Point
		switch dir {
		case guigui.LayoutDirectionHorizontal:
			want = image.Pt(100, 50)
		case guigui.LayoutDirectionVertical:
			want = image.Pt(100, 50)
		}
		if got != want {
			t.Errorf("dir: %v, got: %v, want: %v", dir, got, want)
		}
	}
}

func TestLinearLayoutMeasureFlexibleSizeWithWidgetWithoutConstraints(t *testing.T) {
	l := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       10,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &dummyWidget{
					size: image.Pt(100, 50),
				},
				Size: guigui.FlexibleSize(1),
			},
			{
				Widget: &dummyWidget{
					size: image.Pt(100, 50),
				},
			},
			{
				Widget: &dummyWidget{
					size: image.Pt(120, 60),
				},
				Size: guigui.FlexibleSize(2),
			},
			{
				Widget: &dummyWidget{
					size: image.Pt(140, 70),
				},
				Size: guigui.FlexibleSize(1),
			},
		},
	}
	var context guigui.Context
	got := l.Measure(&context, guigui.Constraints{})
	want := image.Pt(4*140+100+3*10, 70)
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
