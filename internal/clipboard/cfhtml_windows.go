// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

// The CF_HTML clipboard format used on Windows wraps HTML markup in a header
// of "Key:value" lines recording byte offsets into the whole payload:
// StartHTML/EndHTML delimit the document and StartFragment/EndFragment
// delimit the fragment inside it.
// https://learn.microsoft.com/en-us/windows/win32/dataxchg/html-clipboard-format

const (
	cfHTMLPrefix = "<html>\r\n<body>\r\n<!--StartFragment-->"
	cfHTMLSuffix = "<!--EndFragment-->\r\n</body>\r\n</html>"
)

// encodeCFHTML wraps HTML markup in a CF_HTML payload.
func encodeCFHTML(fragment []byte) []byte {
	// Fixed-width offsets keep the header length independent of the values,
	// so the offsets can be computed before formatting.
	const headerFormat = "Version:0.9\r\nStartHTML:%010d\r\nEndHTML:%010d\r\nStartFragment:%010d\r\nEndFragment:%010d\r\n"
	headerLen := len(fmt.Sprintf(headerFormat, 0, 0, 0, 0))
	startHTML := headerLen
	startFragment := startHTML + len(cfHTMLPrefix)
	endFragment := startFragment + len(fragment)
	endHTML := endFragment + len(cfHTMLSuffix)

	var buf bytes.Buffer
	buf.Grow(endHTML)
	fmt.Fprintf(&buf, headerFormat, startHTML, endHTML, startFragment, endFragment)
	buf.WriteString(cfHTMLPrefix)
	buf.Write(fragment)
	buf.WriteString(cfHTMLSuffix)
	return buf.Bytes()
}

// decodeCFHTML extracts the HTML fragment from a CF_HTML payload. When the
// fragment offsets are absent or inconsistent, the document offsets are used
// instead.
func decodeCFHTML(payload []byte) ([]byte, error) {
	offsets := map[string]int{}
	rest := payload
	for len(rest) > 0 && rest[0] != '<' {
		line := rest
		if i := bytes.IndexByte(rest, '\n'); i >= 0 {
			line = rest[:i]
			rest = rest[i+1:]
		} else {
			rest = nil
		}
		line = bytes.TrimSuffix(line, []byte("\r"))
		key, value, ok := bytes.Cut(line, []byte(":"))
		if !ok {
			continue
		}
		// Non-numeric values (Version, SourceURL) are not offsets; skip them.
		n, err := strconv.Atoi(string(bytes.TrimSpace(value)))
		if err != nil {
			continue
		}
		offsets[string(key)] = n
	}

	for _, keys := range [][2]string{
		{"StartFragment", "EndFragment"},
		{"StartHTML", "EndHTML"},
	} {
		start, okStart := offsets[keys[0]]
		end, okEnd := offsets[keys[1]]
		if okStart && okEnd && 0 <= start && start <= end && end <= len(payload) {
			return payload[start:end], nil
		}
	}
	return nil, errors.New("clipboard: invalid CF_HTML payload")
}
