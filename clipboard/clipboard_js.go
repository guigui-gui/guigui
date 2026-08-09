// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import (
	"errors"
	"fmt"
	"syscall/js"
)

// The async clipboard API mandates support for exactly these MIME types.
const (
	mimeText = "text/plain"
	mimeHTML = "text/html"
	mimePNG  = "image/png"
)

// await blocks until the promise settles and returns its resolved value.
func await(promise js.Value) (js.Value, error) {
	type result struct {
		value js.Value
		ok    bool
	}
	ch := make(chan result, 1)
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		var value js.Value
		if len(args) > 0 {
			value = args[0]
		}
		ch <- result{value: value, ok: true}
		return nil
	})
	defer then.Release()
	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		var value js.Value
		if len(args) > 0 {
			value = args[0]
		}
		ch <- result{value: value, ok: false}
		return nil
	})
	defer catch.Release()

	promise.Call("then", then).Call("catch", catch)
	r := <-ch
	if !r.ok {
		if r.value.Type() == js.TypeObject {
			return js.Value{}, js.Error{Value: r.value}
		}
		return js.Value{}, fmt.Errorf("clipboard: promise rejected: %v", r.value)
	}
	return r.value, nil
}

func readContents() (Contents, error) {
	clipboard := js.Global().Get("navigator").Get("clipboard")
	if !clipboard.Truthy() {
		return Contents{}, errors.New("clipboard: navigator.clipboard is unavailable")
	}
	if !clipboard.Get("read").Truthy() {
		return readTextContents(clipboard)
	}

	clipboardItems, err := await(clipboard.Call("read"))
	if err != nil {
		return Contents{}, err
	}

	var contents Contents
	for i := range clipboardItems.Length() {
		clipboardItem := clipboardItems.Index(i)
		types := clipboardItem.Get("types")
		for j := range types.Length() {
			mimeType := types.Index(j).String()
			var dst *[]byte
			switch mimeType {
			case mimeText:
				dst = &contents.Text
			case mimeHTML:
				dst = &contents.HTML
			case mimePNG:
				dst = &contents.Image
			default:
				continue
			}
			if *dst != nil {
				continue
			}
			blob, err := await(clipboardItem.Call("getType", mimeType))
			if err != nil {
				return Contents{}, err
			}
			arrayBuffer, err := await(blob.Call("arrayBuffer"))
			if err != nil {
				return Contents{}, err
			}
			uint8Array := js.Global().Get("Uint8Array").New(arrayBuffer)
			data := make([]byte, uint8Array.Get("length").Int())
			js.CopyBytesToGo(data, uint8Array)
			*dst = data
		}
	}
	return contents, nil
}

// readTextContents reads plain text via readText for environments without
// navigator.clipboard.read.
func readTextContents(clipboard js.Value) (Contents, error) {
	text, err := await(clipboard.Call("readText"))
	if err != nil {
		return Contents{}, err
	}
	return Contents{
		Text: []byte(text.String()),
	}, nil
}

func writeContents(contents Contents) error {
	clipboard := js.Global().Get("navigator").Get("clipboard")
	if !clipboard.Truthy() {
		return errors.New("clipboard: navigator.clipboard is unavailable")
	}
	if !js.Global().Get("ClipboardItem").Truthy() || !clipboard.Get("write").Truthy() {
		return writeTextContents(clipboard, contents)
	}

	representations := js.Global().Get("Object").New()
	setRepresentation := func(mimeType string, data []byte) {
		if data == nil {
			return
		}
		uint8Array := js.Global().Get("Uint8Array").New(len(data))
		js.CopyBytesToJS(uint8Array, data)
		blob := js.Global().Get("Blob").New([]any{uint8Array}, map[string]any{
			"type": mimeType,
		})
		representations.Set(mimeType, blob)
	}
	setRepresentation(mimeText, contents.Text)
	setRepresentation(mimeHTML, contents.HTML)
	setRepresentation(mimePNG, contents.Image)

	clipboardItem := js.Global().Get("ClipboardItem").New(representations)
	if _, err := await(clipboard.Call("write", []any{clipboardItem})); err != nil {
		return err
	}
	return nil
}

// writeTextContents writes the plain-text representation via writeText for
// environments without ClipboardItem. Other representations are dropped.
func writeTextContents(clipboard js.Value, contents Contents) error {
	if contents.Text == nil {
		return errors.New("clipboard: ClipboardItem is unavailable")
	}
	if _, err := await(clipboard.Call("writeText", string(contents.Text))); err != nil {
		return err
	}
	return nil
}
