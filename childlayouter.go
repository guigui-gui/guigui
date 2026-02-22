// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package guigui

import "image"

type ChildLayouter struct {
}

func (c *ChildLayouter) LayoutWidget(widget Widget, bounds image.Rectangle) {
	widget.widgetState().bounds = bounds
}
