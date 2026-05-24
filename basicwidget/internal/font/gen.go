// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

//go:build ignore

package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	if err := xmain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func xmain() error {
	const url = "https://github.com/rsms/inter/releases/download/v4.1/Inter-4.1.zip"
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bs, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	r := bytes.NewReader(bs)
	f, err := zip.NewReader(r, int64(len(bs)))
	if err != nil {
		return err
	}
	for _, f := range f.File {
		const filename = "InterVariable.ttf"
		if f.Name != filename {
			continue
		}
		out, err := os.Create(filename + ".gz")
		if err != nil {
			return err
		}
		defer out.Close()

		w := bufio.NewWriter(out)
		gw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			return err
		}

		r, err := f.Open()
		if err != nil {
			return err
		}
		defer r.Close()

		if _, err := io.Copy(gw, r); err != nil {
			return err
		}
		if err := gw.Close(); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	return nil
}
