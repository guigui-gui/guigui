// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package clipboard

import "syscall/js"

func readAll() (string, error) {
	ch := make(chan string)
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- args[0].String()
		return nil
	})
	defer then.Release()

	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("console").Call("error", "clipboard read failed", args[0])
		close(ch)
		return nil
	})
	defer catch.Release()

	js.Global().Get("navigator").Get("clipboard").Call("readText").Call("then", then).Call("catch", catch)
	return <-ch, nil
}

func writeAll(text string) error {
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

	js.Global().Get("navigator").Get("clipboard").Call("writeText", text).Call("then", then).Call("catch", catch)
	<-ch
	return nil
}
