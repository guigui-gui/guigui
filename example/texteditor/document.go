// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"errors"
	"io"
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

// LoadInto opens the file at path and streams its contents into dst.
// On success the document's path is updated and dirty is cleared.
func (d *Document) LoadInto(path string, dst io.ReaderFrom) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err := dst.ReadFrom(f); err != nil {
		return err
	}
	d.path = path
	d.dirty = false
	return nil
}

// Save streams src to the document's current path. On success dirty is cleared.
func (d *Document) Save(src io.WriterTo) error {
	if d.path == "" {
		return errors.New("no path set; use SaveAs")
	}
	f, err := os.Create(d.path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err := src.WriteTo(f); err != nil {
		return err
	}
	d.dirty = false
	return nil
}

// SaveAs sets the path and streams src to it.
func (d *Document) SaveAs(path string, src io.WriterTo) error {
	d.path = path
	return d.Save(src)
}
