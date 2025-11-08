// Copyright 2025 Hajime Hoshi

package guigui

import "image"

type ChildLayouter struct {
}

func (c *ChildLayouter) LayoutWidget(widget Widget, bounds image.Rectangle) {
	widget.widgetState().bounds = bounds
}
