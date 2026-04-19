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

// StateKeyWriter accumulates a hash of a widget's state.
// Widgets write their state via the Write* methods in [Widget.WriteStateKey];
// the framework reads the resulting hash after the call.
//
// StateKeyWriter implements [io.Writer] as an escape hatch for variable-length
// byte content.
type StateKeyWriter struct {
	h   xxh3.Hasher
	buf [8]byte
}

var _ io.Writer = (*StateKeyWriter)(nil)

func (w *StateKeyWriter) reset() {
	w.h.Reset()
}

func (w *StateKeyWriter) sum128() xxh3.Uint128 {
	return w.h.Sum128()
}

// Write implements [io.Writer].
func (w *StateKeyWriter) Write(p []byte) (int, error) {
	return w.h.Write(p)
}

// WriteBool writes a bool into the writer.
func (w *StateKeyWriter) WriteBool(v bool) {
	if v {
		w.buf[0] = 1
	} else {
		w.buf[0] = 0
	}
	_, _ = w.h.Write(w.buf[:1])
}

// WriteUint8 writes a uint8 into the writer.
func (w *StateKeyWriter) WriteUint8(v uint8) {
	w.buf[0] = v
	_, _ = w.h.Write(w.buf[:1])
}

// WriteUint16 writes a uint16 into the writer.
func (w *StateKeyWriter) WriteUint16(v uint16) {
	binary.LittleEndian.PutUint16(w.buf[:2], v)
	_, _ = w.h.Write(w.buf[:2])
}

// WriteUint32 writes a uint32 into the writer.
func (w *StateKeyWriter) WriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(w.buf[:4], v)
	_, _ = w.h.Write(w.buf[:4])
}

// WriteUint64 writes a uint64 into the writer.
func (w *StateKeyWriter) WriteUint64(v uint64) {
	binary.LittleEndian.PutUint64(w.buf[:8], v)
	_, _ = w.h.Write(w.buf[:8])
}

// is32bit is true on platforms where uint (and uintptr) are 32 bits.
const is32bit = ^uint(0)>>32 == 0

// WriteUint writes a uint into the writer using its native width
// (4 bytes on 32-bit platforms, 8 bytes on 64-bit).
func (w *StateKeyWriter) WriteUint(v uint) {
	if is32bit {
		w.WriteUint32(uint32(v))
	} else {
		w.WriteUint64(uint64(v))
	}
}

// WriteInt8 writes an int8 into the writer.
func (w *StateKeyWriter) WriteInt8(v int8) {
	w.WriteUint8(uint8(v))
}

// WriteInt16 writes an int16 into the writer.
func (w *StateKeyWriter) WriteInt16(v int16) {
	w.WriteUint16(uint16(v))
}

// WriteInt32 writes an int32 into the writer.
func (w *StateKeyWriter) WriteInt32(v int32) {
	w.WriteUint32(uint32(v))
}

// WriteInt64 writes an int64 into the writer.
func (w *StateKeyWriter) WriteInt64(v int64) {
	w.WriteUint64(uint64(v))
}

// WriteInt writes an int into the writer using its native width
// (4 bytes on 32-bit platforms, 8 bytes on 64-bit).
func (w *StateKeyWriter) WriteInt(v int) {
	if is32bit {
		w.WriteInt32(int32(v))
	} else {
		w.WriteInt64(int64(v))
	}
}

// WriteFloat32 writes a float32 into the writer.
func (w *StateKeyWriter) WriteFloat32(v float32) {
	w.WriteUint32(math.Float32bits(v))
}

// WriteFloat64 writes a float64 into the writer.
func (w *StateKeyWriter) WriteFloat64(v float64) {
	w.WriteUint64(math.Float64bits(v))
}

// WriteString writes a string into the writer. The length is included so that
// concatenations hash distinctly (e.g. "ab"+"cd" vs "abc"+"d").
func (w *StateKeyWriter) WriteString(s string) {
	w.WriteUint64(uint64(len(s)))
	_, _ = io.WriteString(&w.h, s)
}

// WriteWidget writes the identity of a [Widget] into the writer.
// A nil widget hashes to zero.
func (w *StateKeyWriter) WriteWidget(widget Widget) {
	var p uintptr
	if widget != nil {
		p = uintptr(unsafe.Pointer(widget.widgetState()))
	}
	if is32bit {
		w.WriteUint32(uint32(p))
	} else {
		w.WriteUint64(uint64(p))
	}
}
