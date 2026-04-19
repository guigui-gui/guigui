// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package guigui

import (
	"encoding/binary"
	"io"
	"math"
	"unsafe"

	"github.com/zeebo/xxh3"
)

// BuildKeyHasher accumulates a hash of a widget's state.
// Widgets write their state via the Write* methods in [Widget.BuildKey];
// the framework reads the resulting hash after the call.
//
// BuildKeyHasher implements [io.Writer] as an escape hatch for variable-length
// byte content.
type BuildKeyHasher struct {
	h   xxh3.Hasher
	buf [8]byte
}

var _ io.Writer = (*BuildKeyHasher)(nil)

func (h *BuildKeyHasher) reset() {
	h.h.Reset()
}

func (h *BuildKeyHasher) sum128() xxh3.Uint128 {
	return h.h.Sum128()
}

// Write implements [io.Writer].
func (h *BuildKeyHasher) Write(p []byte) (int, error) {
	return h.h.Write(p)
}

// WriteBool writes a bool into the hasher.
func (h *BuildKeyHasher) WriteBool(v bool) {
	if v {
		h.buf[0] = 1
	} else {
		h.buf[0] = 0
	}
	_, _ = h.h.Write(h.buf[:1])
}

// WriteUint8 writes a uint8 into the hasher.
func (h *BuildKeyHasher) WriteUint8(v uint8) {
	h.buf[0] = v
	_, _ = h.h.Write(h.buf[:1])
}

// WriteUint16 writes a uint16 into the hasher.
func (h *BuildKeyHasher) WriteUint16(v uint16) {
	binary.LittleEndian.PutUint16(h.buf[:2], v)
	_, _ = h.h.Write(h.buf[:2])
}

// WriteUint32 writes a uint32 into the hasher.
func (h *BuildKeyHasher) WriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(h.buf[:4], v)
	_, _ = h.h.Write(h.buf[:4])
}

// WriteUint64 writes a uint64 into the hasher.
func (h *BuildKeyHasher) WriteUint64(v uint64) {
	binary.LittleEndian.PutUint64(h.buf[:8], v)
	_, _ = h.h.Write(h.buf[:8])
}

// is32bit is true on platforms where uint (and uintptr) are 32 bits.
const is32bit = ^uint(0)>>32 == 0

// WriteUint writes a uint into the hasher using its native width
// (4 bytes on 32-bit platforms, 8 bytes on 64-bit).
func (h *BuildKeyHasher) WriteUint(v uint) {
	if is32bit {
		h.WriteUint32(uint32(v))
	} else {
		h.WriteUint64(uint64(v))
	}
}

// WriteInt8 writes an int8 into the hasher.
func (h *BuildKeyHasher) WriteInt8(v int8) {
	h.WriteUint8(uint8(v))
}

// WriteInt16 writes an int16 into the hasher.
func (h *BuildKeyHasher) WriteInt16(v int16) {
	h.WriteUint16(uint16(v))
}

// WriteInt32 writes an int32 into the hasher.
func (h *BuildKeyHasher) WriteInt32(v int32) {
	h.WriteUint32(uint32(v))
}

// WriteInt64 writes an int64 into the hasher.
func (h *BuildKeyHasher) WriteInt64(v int64) {
	h.WriteUint64(uint64(v))
}

// WriteInt writes an int into the hasher using its native width
// (4 bytes on 32-bit platforms, 8 bytes on 64-bit).
func (h *BuildKeyHasher) WriteInt(v int) {
	if is32bit {
		h.WriteInt32(int32(v))
	} else {
		h.WriteInt64(int64(v))
	}
}

// WriteFloat32 writes a float32 into the hasher.
func (h *BuildKeyHasher) WriteFloat32(v float32) {
	h.WriteUint32(math.Float32bits(v))
}

// WriteFloat64 writes a float64 into the hasher.
func (h *BuildKeyHasher) WriteFloat64(v float64) {
	h.WriteUint64(math.Float64bits(v))
}

// WriteString writes a string into the hasher. The length is included so that
// concatenations hash distinctly (e.g. "ab"+"cd" vs "abc"+"d").
func (h *BuildKeyHasher) WriteString(s string) {
	h.WriteUint64(uint64(len(s)))
	_, _ = io.WriteString(&h.h, s)
}

// WriteWidget writes the identity of a [Widget] into the hasher.
// A nil widget hashes to zero.
func (h *BuildKeyHasher) WriteWidget(w Widget) {
	var p uintptr
	if w != nil {
		p = uintptr(unsafe.Pointer(w.widgetState()))
	}
	if is32bit {
		h.WriteUint32(uint32(p))
	} else {
		h.WriteUint64(uint64(p))
	}
}
