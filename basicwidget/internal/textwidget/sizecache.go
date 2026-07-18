// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package textwidget

import (
	"github.com/guigui-gui/guigui/basicwidget/internal/textutil"
)

type textSizeCacheKey int

func newTextSizeCacheKey(wrapMode textutil.WrapMode, bold bool) textSizeCacheKey {
	key := textSizeCacheKey(wrapMode) & 0x3
	if bold {
		key |= 1 << 2
	}
	return key
}

type cachedTextWidthEntry struct {
	// 0 indicates that the entry is invalid.
	constraintWidth int

	width int
}

type cachedTextHeightEntry struct {
	// 0 indicates that the entry is invalid.
	constraintWidth int

	height int
}

// textSizeCache memoizes measured text widths and heights per
// [textSizeCacheKey] and constraint width, with move-to-front replacement
// within each key's entries.
type textSizeCache struct {
	widths  [8][4]cachedTextWidthEntry
	heights [8][4]cachedTextHeightEntry

	// defaultTabWidth is the measured advance of the default tab string. The
	// zero value means it has not been measured yet.
	defaultTabWidth float64
}

// width returns the cached width for key and constraintWidth.
func (c *textSizeCache) width(key textSizeCacheKey, constraintWidth int) (int, bool) {
	for i := range c.widths[key] {
		entry := &c.widths[key][i]
		if entry.constraintWidth == 0 {
			continue
		}
		if entry.constraintWidth != constraintWidth {
			continue
		}
		if i != 0 {
			e := *entry
			copy(c.widths[key][1:i+1], c.widths[key][:i])
			c.widths[key][0] = e
		}
		return c.widths[key][0].width, true
	}
	return 0, false
}

// setWidth records the width for key and constraintWidth as the most recent
// entry, evicting the oldest one.
func (c *textSizeCache) setWidth(key textSizeCacheKey, constraintWidth, width int) {
	copy(c.widths[key][1:], c.widths[key][:])
	c.widths[key][0] = cachedTextWidthEntry{
		constraintWidth: constraintWidth,
		width:           width,
	}
}

// height returns the cached height for key and constraintWidth.
func (c *textSizeCache) height(key textSizeCacheKey, constraintWidth int) (int, bool) {
	for i := range c.heights[key] {
		entry := &c.heights[key][i]
		if entry.constraintWidth == 0 {
			continue
		}
		if entry.constraintWidth != constraintWidth {
			continue
		}
		if i != 0 {
			e := *entry
			copy(c.heights[key][1:i+1], c.heights[key][:i])
			c.heights[key][0] = e
		}
		return c.heights[key][0].height, true
	}
	return 0, false
}

// setHeight records the height for key and constraintWidth as the most recent
// entry, evicting the oldest one.
func (c *textSizeCache) setHeight(key textSizeCacheKey, constraintWidth, height int) {
	copy(c.heights[key][1:], c.heights[key][:])
	c.heights[key][0] = cachedTextHeightEntry{
		constraintWidth: constraintWidth,
		height:          height,
	}
}

// reset invalidates all cached sizes and the default tab width.
func (c *textSizeCache) reset() {
	clear(c.widths[:])
	clear(c.heights[:])
	c.defaultTabWidth = 0
}
