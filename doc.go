// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package guigui provides a GUI framework for Go built on top of Ebitengine.
//
// # Widget lifecycle
//
// The core of guigui is the [Widget] interface. All UI components implement this interface
// by embedding [DefaultWidget] in their structs.
//
// The framework guarantees the following about a few of the widget methods:
//
//   - [Widget.Build] constructs the child widget tree. When Build is called, the widget's own
//     bounds are not determined yet (so [WidgetBounds] is not passed as an argument), and the
//     child tree is not determined yet either.
//   - [Widget.Layout] positions and sizes children. When Layout is called, the widget's own bounds
//     and its parent's bounds are determined, but its children's bounds are not yet determined.
//   - [Widget.Tick] is invoked at the application's TPS (60 times per second by default,
//     or whatever TPS the user has configured).
//   - [Widget.HandlePointingInput] is invoked in post-order (children before their parent),
//     per layer from top to bottom. This lets an inner or higher-layer widget consume a pointing
//     event before its ancestors or lower layers see it.
//   - [Widget.HandleButtonInput] is invoked with the same post-order, top-to-bottom-layer
//     traversal, but only on a subset of widgets: roughly, a widget that is focused, has a focused
//     descendant, has a focused or button-input-receptive ancestor, or is itself
//     button-input-receptive (see [Context.SetButtonInputReceptive]). Disabled or hidden widgets
//     are skipped. See [Widget.HandleButtonInput] for the exact conditions.
//   - [Widget.Draw] is invoked in pre-order (parent before its children), per layer from bottom
//     to top, so children and higher layers are rendered on top of their parents and lower layers.
//
// All other aspects — such as when and how often each method is called — are implementation
// details that the framework may change.
//
// # Running an application
//
// Use [Run] to start an application with a root widget:
//
//	type Root struct {
//		guigui.DefaultWidget
//		// ...
//	}
//
//	func main() {
//		if err := guigui.Run(&Root{}, &guigui.RunOptions{
//			Title: "My App",
//		}); err != nil {
//			log.Fatal(err)
//		}
//	}
package guigui
