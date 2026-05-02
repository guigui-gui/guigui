// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"io"

	"github.com/guigui-gui/guigui/basicwidget"
)

// editor wraps [basicwidget.TextInput] so that *editor satisfies
// [io.WriterTo] and [io.ReaderFrom]. The document layer can then stream
// through stdlib interfaces without basicwidget itself having to use those
// names.
type editor struct {
	basicwidget.TextInput
}

func (e *editor) WriteTo(w io.Writer) (int64, error) {
	return e.WriteValueTo(w)
}

func (e *editor) ReadFrom(r io.Reader) (int64, error) {
	return e.ReadValueFrom(r)
}
