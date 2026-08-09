// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package clipboard_test

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/guigui-gui/guigui/clipboard"
)

func cfHTMLHeaderOffset(t *testing.T, payload []byte, key string) int {
	t.Helper()
	for line := range strings.SplitSeq(string(payload), "\r\n") {
		v, ok := strings.CutPrefix(line, key+":")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("invalid %s value %q: %v", key, v, err)
		}
		return n
	}
	t.Fatalf("header field %s not found", key)
	return 0
}

func TestEncodeCFHTML(t *testing.T) {
	fragment := []byte(`Hello, <b>World</b> &amp; more`)
	payload := clipboard.EncodeCFHTML(fragment)

	if !bytes.HasPrefix(payload, []byte("Version:0.9\r\n")) {
		t.Errorf("payload does not start with a Version line: %q", payload)
	}

	startHTML := cfHTMLHeaderOffset(t, payload, "StartHTML")
	endHTML := cfHTMLHeaderOffset(t, payload, "EndHTML")
	startFragment := cfHTMLHeaderOffset(t, payload, "StartFragment")
	endFragment := cfHTMLHeaderOffset(t, payload, "EndFragment")

	if got, want := endHTML, len(payload); got != want {
		t.Errorf("EndHTML: got %d, want %d", got, want)
	}
	if payload[startHTML] != '<' {
		t.Errorf("StartHTML does not point at markup: %q", payload[startHTML:])
	}
	if got, want := string(payload[startFragment:endFragment]), string(fragment); got != want {
		t.Errorf("fragment range: got %q, want %q", got, want)
	}
	const startMarker = "<!--StartFragment-->"
	if got, want := string(payload[startFragment-len(startMarker):startFragment]), startMarker; got != want {
		t.Errorf("before StartFragment: got %q, want %q", got, want)
	}
	const endMarker = "<!--EndFragment-->"
	if got, want := string(payload[endFragment:endFragment+len(endMarker)]), endMarker; got != want {
		t.Errorf("at EndFragment: got %q, want %q", got, want)
	}
}

func TestEncodeDecodeCFHTMLRoundTrip(t *testing.T) {
	for _, fragment := range []string{
		"",
		"plain",
		"<b>bold</b> and <i>italic</i>",
		"non-ASCII: こんにちは 🎉",
	} {
		t.Run(fragment, func(t *testing.T) {
			got, err := clipboard.DecodeCFHTML(clipboard.EncodeCFHTML([]byte(fragment)))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != fragment {
				t.Errorf("got %q, want %q", got, fragment)
			}
		})
	}
}

func TestDecodeCFHTMLVariableWidthOffsets(t *testing.T) {
	// Producers are not required to zero-pad offsets; some also include a
	// SourceURL line. Build such a payload with a fixed 4-digit width.
	const headerFormat = "Version:1.0\r\nStartHTML:%04d\r\nEndHTML:%04d\r\nStartFragment:%04d\r\nEndFragment:%04d\r\nSourceURL:https://example.com/\r\n"
	const fragment = "<span>fragment</span>"
	const prefix = "<html><body><!--StartFragment-->"
	const suffix = "<!--EndFragment--></body></html>"
	headerLen := len(fmt.Sprintf(headerFormat, 0, 0, 0, 0))
	startFragment := headerLen + len(prefix)
	endFragment := startFragment + len(fragment)
	endHTML := endFragment + len(suffix)
	payload := fmt.Sprintf(headerFormat, headerLen, endHTML, startFragment, endFragment) + prefix + fragment + suffix

	got, err := clipboard.DecodeCFHTML([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != fragment {
		t.Errorf("got %q, want %q", got, fragment)
	}
}

func TestDecodeCFHTMLDocumentFallback(t *testing.T) {
	// Without fragment offsets, the document offsets delimit the result.
	const document = "<html><body>doc</body></html>"
	const headerFormat = "Version:0.9\r\nStartHTML:%04d\r\nEndHTML:%04d\r\n"
	headerLen := len(fmt.Sprintf(headerFormat, 0, 0))
	payload := fmt.Sprintf(headerFormat, headerLen, headerLen+len(document)) + document

	got, err := clipboard.DecodeCFHTML([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != document {
		t.Errorf("got %q, want %q", got, document)
	}
}

func TestDecodeCFHTMLInvalid(t *testing.T) {
	for name, payload := range map[string]string{
		"empty":            "",
		"no header":        "<html><body>hi</body></html>",
		"non-numeric":      "Version:0.9\r\nStartHTML:x\r\nEndHTML:y\r\n<html></html>",
		"out of range":     "Version:0.9\r\nStartFragment:10\r\nEndFragment:9999\r\n<html></html>",
		"inverted range":   "Version:0.9\r\nStartFragment:20\r\nEndFragment:10\r\n<html></html>",
		"negative offsets": "Version:0.9\r\nStartFragment:-1\r\nEndFragment:-1\r\n<html></html>",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := clipboard.DecodeCFHTML([]byte(payload)); err == nil {
				t.Errorf("expected an error for %q", payload)
			}
		})
	}
}
