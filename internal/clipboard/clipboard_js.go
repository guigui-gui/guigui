// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import "syscall/js"

func readAll() ([]byte, error) {
	ch := make(chan []byte)
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- []byte(args[0].String())
		return nil
	})
	defer then.Release()

	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("console").Call("error", "clipboard read failed", args[0])
		close(ch)
		return nil
	})
	defer catch.Release()

	// TODO: Use read.
	js.Global().Get("navigator").Get("clipboard").Call("readText").Call("then", then).Call("catch", catch)
	return <-ch, nil
}

func writeAll(text []byte) error {
	ch := make(chan struct{})
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		close(ch)
		return nil
	})
	defer then.Release()

	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("console").Call("error", "clipboard write failed", args[0])
		close(ch)
		return nil
	})
	defer catch.Release()

	// TODO: Use write.
	js.Global().Get("navigator").Get("clipboard").Call("writeText", string(text)).Call("then", then).Call("catch", catch)
	<-ch
	return nil
}
