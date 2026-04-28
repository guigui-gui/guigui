// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"errors"
	"os"
	"path/filepath"
)

type Document struct {
	path  string
	dirty bool
}

func (d *Document) Path() string {
	return d.path
}

func (d *Document) IsDirty() bool {
	return d.dirty
}

func (d *Document) MarkDirty() {
	d.dirty = true
}

func (d *Document) MarkClean() {
	d.dirty = false
}

func (d *Document) DisplayName() string {
	if d.path == "" {
		return "Untitled"
	}
	return filepath.Base(d.path)
}

func (d *Document) New() {
	d.path = ""
	d.dirty = false
}

// Load reads the file at path. The caller is responsible for installing
// the returned text into the editor.
func (d *Document) Load(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	d.path = path
	d.dirty = false
	return string(b), nil
}

func (d *Document) Save(text string) error {
	if d.path == "" {
		return errors.New("no path set; use SaveAs")
	}
	if err := os.WriteFile(d.path, []byte(text), 0o644); err != nil {
		return err
	}
	d.dirty = false
	return nil
}

func (d *Document) SaveAs(path, text string) error {
	d.path = path
	return d.Save(text)
}
