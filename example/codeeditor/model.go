// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

// initialCode is the sample Go source code the editor starts with.
const initialCode = `// Package main demonstrates syntax highlighting.
package main

import "fmt"

const answer = 42

type Greeter struct {
	name string
}

// Greet returns a friendly greeting.
func (g Greeter) Greet() string {
	return fmt.Sprintf("Hello, %s!", g.name)
}

func main() {
	g := Greeter{name: "gopher"}
	for i := range 3 {
		fmt.Println(g.Greet(), i*answer)
	}
}
`

// Model holds the application state: the source code being edited. The
// highlighting styles are not part of the state; the view derives them from
// the code on each build.
type Model struct {
	code   string
	inited bool
}

// Code returns the source code being edited.
func (m *Model) Code() string {
	if !m.inited {
		return initialCode
	}
	return m.code
}

// SetCode sets the source code being edited.
func (m *Model) SetCode(code string) {
	m.code = code
	m.inited = true
}
