// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

// clickCounter tracks consecutive clicks at the same text position so that
// double- and triple-clicks can be distinguished from repeated single clicks.
type clickCounter struct {
	count         int
	lastTick      int64
	lastTextIndex int
}

// click records a click at textIndex on tick and returns the updated
// consecutive-click count. A left click within [doubleClickLimitInTicks] of
// the previous click at the same text index increments the count; any other
// click resets it to 1.
func (c *clickCounter) click(tick int64, textIndex int, leftClick bool) int {
	if leftClick && tick-c.lastTick < int64(doubleClickLimitInTicks()) && c.lastTextIndex == textIndex {
		c.count++
	} else {
		c.count = 1
	}
	c.lastTick = tick
	c.lastTextIndex = textIndex
	return c.count
}

// reset clears the consecutive-click count while recording a click at
// textIndex on tick, so a following click is not treated as a double- or
// triple-click.
func (c *clickCounter) reset(tick int64, textIndex int) {
	c.count = 0
	c.lastTick = tick
	c.lastTextIndex = textIndex
}
