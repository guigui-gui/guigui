// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"errors"
	"os"
	"path/filepath"
)

type Document struct {
	path string
	// Holding the entire file as a single string is fine for an example
	// but scales poorly: every edit reallocates and copies the whole
	// buffer, and dirty tracking keeps two full copies (text + saved).
	// A real implementation would use a piece table (or similar
	// structure) so edits and dirty tracking are O(edit-size) rather
	// than O(file-size).
	text  string
	saved string
}

func (d *Document) Path() string {
	return d.path
}

func (d *Document) Text() string {
	return d.text
}

func (d *Document) IsDirty() bool {
	return d.text != d.saved
}

func (d *Document) SetText(text string) {
	d.text = text
}

func (d *Document) DisplayName() string {
	if d.path == "" {
		return "Untitled"
	}
	return filepath.Base(d.path)
}

func (d *Document) New() {
	d.path = ""
	d.text = ""
	d.saved = ""
}

func (d *Document) Load(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	d.path = path
	d.text = string(b)
	d.saved = d.text
	return nil
}

func (d *Document) Save() error {
	if d.path == "" {
		return errors.New("no path set; use SaveAs")
	}
	if err := os.WriteFile(d.path, []byte(d.text), 0o644); err != nil {
		return err
	}
	d.saved = d.text
	return nil
}

func (d *Document) SaveAs(path string) error {
	d.path = path
	return d.Save()
}
