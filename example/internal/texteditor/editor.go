// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package texteditor

import (
	"io"

	"github.com/guigui-gui/guigui/basicwidget"
)

// Editor wraps [basicwidget.TextInput] so that *Editor satisfies
// [io.WriterTo] and [io.ReaderFrom]. The document layer can then stream
// through stdlib interfaces without basicwidget itself having to use those
// names.
type Editor struct {
	basicwidget.TextInput
}

func (e *Editor) WriteTo(w io.Writer) (int64, error) {
	return e.WriteValueTo(w)
}

func (e *Editor) ReadFrom(r io.Reader) (int64, error) {
	return e.ReadValueFrom(r)
}
