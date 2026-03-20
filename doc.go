// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

// Package guigui provides a GUI framework for Go built on top of Ebitengine.
//
// # Widget lifecycle
//
// The core of guigui is the [Widget] interface. All UI components implement this interface
// by embedding [DefaultWidget] in their structs.
//
// The framework is built on Ebitengine's game loop. During each Update (tick),
// the following methods are called in order:
//
//  1. Build+Layout phase (1) - [Widget.Build] constructs the child widget tree (pre-order; skipped if unnecessary),
//     then [Widget.Layout] positions and sizes children within the widget's bounds (pre-order; skipped if unnecessary).
//     This may repeat if build or layout triggers further rebuild/relayout requests, up to a fixed limit.
//  2. Input phase - [Widget.HandlePointingInput] handles mouse and touch input (skipped when no relevant pointing input is active), and [Widget.HandleButtonInput] handles keyboard and gamepad input (post-order, per layer from top to bottom).
//  3. Build+Layout phase (2) - Same as phase 1, reflecting the latest state after input handling.
//  4. Tick phase - [Widget.Tick] updates widget state (pre-order).
//
// During each Draw (frame), the following method is called:
//
//   - [Widget.Draw] — renders the widget (pre-order, per layer from bottom to top; only when necessary).
//
// [Widget.Measure] and [Widget.Env] are called on demand by other widgets or the framework.
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
