---
name: using-guigui
description: >
  Use this skill when writing or modifying GUI code with the Guigui framework
  (github.com/guigui-gui/guigui): creating or editing a widget, composing a
  layout, wiring buttons / text inputs / lists, passing shared state down the
  widget tree, handling pointer or keyboard input, or figuring out why a change
  to a field does not show up on screen. Covers the widget lifecycle
  (Build / Layout / Measure / Draw / input), LinearLayout sizing, Env-based
  dependency injection, the event-key callback pattern, dynamic child lists, and
  when state changes trigger a rebuild. Not for general Ebitengine rendering
  questions that do not involve Guigui widgets.
---

# Using Guigui

Guigui is an immediate-mode-*inspired* GUI framework for Go on top of Ebitengine.
"Inspired" is the key word: widgets are **retained** Go structs that you own and
keep across ticks, but their child tree and presentation are **reconstructed**
by re-running `Build` whenever relevant state changes — the way Compose or
SwiftUI re-run a view function. You hold the state in struct fields; the
framework decides when to rebuild, lay out, and redraw.

Read this whole file before writing widget code; the lifecycle rules below are
the part people get wrong.

> **Freshness.** Verified against Guigui at commit `4ebb3fc7` (2026-06-25).
> Guigui is alpha and its API may change — if anything here disagrees with the
> source, **the source wins**: trust `*.go` in the module root and the programs
> under `example/` over this file, and update this skill when you find drift.

## The mental model

- A **widget** is a struct that embeds `guigui.DefaultWidget` and overrides the
  methods it needs. `DefaultWidget` supplies no-op defaults for every method, so
  override only what matters.
- A widget must work **from its zero value** — declare children as plain
  (non-pointer) fields and reference them as `&w.child`. Do not allocate them in
  a constructor; `var w Foo` must already be usable.
- The framework runs two decoupled cycles. Each **tick** (Ebitengine's `Update`,
  at the app's TPS — 60×/sec by default, or whatever TPS is configured) settles
  the tree: it may run **Build** (reconstruct the child tree), **Layout**
  (position the children), and the **`Handle*Input`** methods an *indeterminate*
  number of times, interleaved, until state stops changing — then calls **Tick**
  exactly once. Rely only on these guarantees: within a pass Build precedes
  Layout, and Tick runs once, last. Do not assume a fixed count of Build, Layout,
  or input passes per tick, nor that an input handler runs only once before
  `Tick`. Each **frame** (Ebitengine's `Draw`, at the display's refresh rate)
  does **Draw**. Ticks and frames are *not* one-to-one — there may be more or
  fewer frames than ticks — so put per-step logic in `Tick`, never in `Draw`.
- You never invoke a widget's framework-driven methods (`Build`, `Layout`,
  `Tick`, `Draw`, the `Handle*Input` methods, `Env`, …) yourself — you set fields
  and register handlers, and the framework calls back. The one exception is
  **`Measure`**: a parent or composite widget calls a child's `Measure` directly
  to size it (it is what `LinearLayout` and the shared `layout()` helper do).

## Minimal application

```go
package main

import (
	"image"
	"log"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Root struct {
	guigui.DefaultWidget

	background basicwidget.Background
	hello      basicwidget.Text
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.hello)

	r.hello.SetValue("Hello, Guigui")
	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())
	layouter.LayoutWidget(&r.hello, widgetBounds.Bounds())
}

func main() {
	if err := guigui.Run(&Root{}, &guigui.RunOptions{
		Title:         "Example",
		WindowMinSize: image.Pt(480, 320),
	}); err != nil {
		log.Fatal(err)
	}
}
```

`guigui.Run(root, *RunOptions)` starts the app. `RunOptions` carries `Title`,
`WindowSize` / `WindowMinSize` / `WindowMaxSize`, `AppScale`, and an optional
`RunGameOptions` passed through to Ebitengine.

## The Widget interface

Embed `guigui.DefaultWidget`, then override as needed:

| Method | Signature | Override when |
|---|---|---|
| `Build` | `Build(*Context, *ChildAdder) error` | Almost always — declare and configure children. |
| `Layout` | `Layout(*Context, *WidgetBounds, *ChildLayouter)` | You have children to position (i.e. almost always). |
| `Measure` | `Measure(*Context, Constraints) image.Point` | The widget has an intrinsic preferred size (list rows, leaf widgets). |
| `HandlePointingInput` | `HandlePointingInput(*Context, *WidgetBounds) HandleInputResult` | Custom mouse/touch handling. |
| `HandleButtonInput` | `HandleButtonInput(*Context, *WidgetBounds) HandleInputResult` | Custom keyboard/gamepad handling. |
| `Env` | `Env(*Context, EnvKey, *EnvSource) (any, bool)` | The widget provides shared values to descendants. |
| `WriteStateKey` | `WriteStateKey(*StateKeyWriter)` | State changes outside input/events must trigger a rebuild (see below). |
| `Tick` | `Tick(*Context, *WidgetBounds) error` | Per-tick updates (animation, timers); runs at the app's TPS. |
| `Draw` | `Draw(*Context, *WidgetBounds, *ebiten.Image)` | Custom rendering (most widgets compose children instead). |
| `CursorShape` | `CursorShape(*Context, *WidgetBounds) (ebiten.CursorShapeType, bool)` | The widget wants a non-default cursor. |

### Build vs. Layout — the rule that matters most

- **`Build` runs before any bounds are known.** Use it to add children
  (`adder.AddWidget(&w.child)`) and to push current state into them
  (`SetText`, `SetEnabled`, registering handlers). `Build` is re-run on every
  rebuild, so it must be **idempotent** — write it as "given my current state,
  configure my children," not "do this once."
- **Conventional order inside `Build`: add first, configure second.** Call all
  the `adder.AddWidget(...)` you need, *then* push state into the children with
  `Set*` / `On*`. Configuring a child you did not add is harmless, so you do not
  need to guard every setter behind the same condition as its `AddWidget` —
  unconditionally configuring for simplicity is fine and idiomatic.
- **Only add children you actually use this rebuild.** Skip the `AddWidget` for a
  child that is not currently shown (e.g. the contents of a closed popup, a
  collapsed panel, a hidden tab). A widget that is not added is not laid out,
  drawn, or sent input, which is cheaper than adding it and hiding it. `popup`
  is the canonical example — it only adds its background/shadow/content while it
  is opening or open, yet still runs their setters unconditionally afterward.
- **`Layout` runs after `Build`, with bounds known.** Use it only to position
  children via `layouter.LayoutWidget(child, bounds)` or a layout helper. Do not
  read bounds in `Build`; do not configure widget content in `Layout`.

`widgetBounds.Bounds()` gives **this** widget's own rectangle inside `Layout`,
input, `Tick`, and `Draw`. A `WidgetBounds` only ever describes the widget it
was handed to — there is no API to ask it for another widget's bounds. If a
parent needs to size or place a child relative to a sibling, drive that from the
layout (`LinearLayout` / `layouter`), not by trying to read the sibling's
rectangle.

## Layout with LinearLayout

`LinearLayout` is the workhorse. Build a slice of `LinearLayoutItem`, then call
`LayoutWidgets`. Reuse the backing slice across ticks with `slices.Delete(s, 0,
len(s))` to avoid per-tick allocation (the framework calls `Layout` often).

```go
func (w *Panel) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context) // scale-aware base unit; size things in multiples of it

	w.items = slices.Delete(w.items, 0, len(w.items))
	w.items = append(w.items,
		guigui.LinearLayoutItem{Widget: &w.header, Size: guigui.FixedSize(u)},
		guigui.LinearLayoutItem{Widget: &w.body, Size: guigui.FlexibleSize(1)}, // takes remaining space
		guigui.LinearLayoutItem{Widget: &w.footer},                            // zero Size = intrinsic (its Measure)
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     w.items,
		Gap:       u / 2,
		Padding:   guigui.Padding{Start: u / 2, Top: u / 2, End: u / 2, Bottom: u / 2},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
```

Sizing per item (`LinearLayoutItem.Size`):

- `guigui.FixedSize(px)` — exact pixel length along the layout direction.
- `guigui.FlexibleSize(weight)` — shares the leftover space proportionally to
  the weight (after fixed/intrinsic items are placed). A bare
  `LinearLayoutItem{Size: guigui.FlexibleSize(1)}` with no `Widget` is a spacer.
- **Zero `Size{}`** — the item is sized by the widget's own `Measure`
  (intrinsic size).

Nesting: an item can carry a sub-layout instead of a widget via
`LinearLayoutItem{Size: ..., Layout: &subLayout}` where `subLayout` is another
`guigui.LinearLayout`. The cross-axis always fills the available extent.

Size things in multiples of `basicwidget.UnitSize(context)` rather than raw
pixels so layouts scale correctly on Hi-DPI and at different app scales.

### Share one layout between `Layout` and `Measure`

A composite widget usually needs the *same* `LinearLayout` in two places:
`Layout` feeds it to `LayoutWidgets`, and `Measure` asks it for a size. Build it
once in an unexported helper — conventionally named `layout` — and call that
helper from both. The structure then has a single source of truth, and `Measure`
can never drift from what `Layout` actually does:

```go
func (w *Field) layout(context *guigui.Context) guigui.LinearLayout {
	u := basicwidget.UnitSize(context)
	w.items = slices.Delete(w.items, 0, len(w.items))
	w.items = append(w.items,
		guigui.LinearLayoutItem{Widget: &w.input, Size: guigui.FlexibleSize(1)},
		guigui.LinearLayoutItem{Widget: &w.button, Size: guigui.FixedSize(u)},
	)
	return guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Gap:       u / 4,
		Items:     w.items,
	}
}

func (w *Field) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	w.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (w *Field) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return w.layout(context).Measure(context, constraints)
}
```

`LinearLayout` is a plain value, so returning one by value is cheap; the helper
still reuses the `items` slice across ticks as before.

## Composition and built-in widgets

Compose by embedding child widgets as fields and adding them in `Build`. The
`basicwidget` package provides the standard catalog — `Background`, `Button`,
`Text`, `TextInput`, `NumberInput`, `Checkbox`, `Toggle`, `RadioButton`,
`Slider`, `Select`/`Combobox`, `List[T]`, `Table[T]`, `Panel`, `Popup`,
`PopupMenu`, `ContextMenuArea[T]`, `SegmentedControl`, `Form`, `Image`,
`Divider`, `Expander`, and more. Configure them with their `Set*` methods in
`Build` and subscribe to their `On*` callbacks.

```go
r.createButton.SetText("Create")
r.createButton.OnUp(func(context *guigui.Context) {
	r.addItem(r.textInput.Value())
})
context.SetEnabled(&r.createButton, r.canAdd())
```

For the full list and runnable demos, read `basicwidget/` and the programs under
`example/` (start with `example/counter` and `example/todo`; `example/gallery`
exercises most widgets).

## Passing shared state down: Env

`Env` is Guigui's dependency injection. A value provided by an ancestor is
visible to every descendant without threading it through constructors. Generate
a key once at package scope:

```go
var envKeyModel = guigui.GenerateEnvKey()

// Ancestor provides the value:
func (r *Root) Env(context *guigui.Context, key guigui.EnvKey, source *guigui.EnvSource) (any, bool) {
	switch key {
	case envKeyModel:
		return &r.model, true
	default:
		return nil, false
	}
}

// Any descendant reads it (typically in Build):
func (c *Child) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	v, ok := context.Env(c, envKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)
	// ... use model to configure children ...
	return nil
}
```

`context.Env(self, key)` walks up the parent chain calling each ancestor's `Env`
until one returns `true`. Use it for app-wide models / view-state; use explicit
fields and setters for parent→direct-child wiring.

`Env` is not only for your own values — `basicwidget` itself uses it to push
presentation state down to descendant widgets. For example a `List` advertises
its item color scheme through `basicwidget.EnvKeyListItemColorType`, and a row
inside the list reads it with `context.Env(self, basicwidget.EnvKeyListItemColorType)`
to draw itself correctly. So when composing a custom widget that lives inside a
built-in container, check whether that container exposes an `EnvKey…` you should
honor.

## Sending events up: the event-key pattern

A child should not know its parent's type. To notify the parent, declare an
`EventKey`, expose an `On…` setter, and dispatch:

```go
var eventItemDeleted = guigui.GenerateEventKey()

func (w *Item) OnDeleted(f func(context *guigui.Context, id int)) {
	guigui.SetEventHandler(w, eventItemDeleted, f)
}

func (w *Item) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&w.deleteButton)
	w.deleteButton.SetText("Delete")
	w.deleteButton.OnUp(func(context *guigui.Context) {
		guigui.DispatchEvent(w, eventItemDeleted, w.id)
	})
	return nil
}
```

The parent calls `child.OnDeleted(func(context, id){ ... })` during its own
`Build`. Handlers are **cleared and re-registered on every rebuild**, so always
(re)register inside `Build`. This is exactly how `basicwidget` buttons expose
`OnUp`.

## Dynamic lists of children

For a variable number of children, use `guigui.WidgetSlice[*T]`: set its length
to match the data, then add and configure each element.

```go
func (l *List) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	l.rows.SetLen(l.model.Count())
	for i := range l.rows.Len() {
		adder.AddWidget(l.rows.At(i))
	}
	for i := range l.model.Count() {
		item := l.model.At(i)
		l.rows.At(i).SetText(item.Text)
		l.rows.At(i).OnActivated(func(context *guigui.Context) {
			l.model.Activate(item.ID)
		})
	}
	return nil
}
```

Use `WidgetSlice`, **not** a hand-rolled `[]Row` you `append` to. A widget's
identity is the address of its embedded `DefaultWidget`, and Guigui forbids
moving or copying a widget by value — it panics with *"illegal use of
DefaultWidget copied by value"*. Appending to a value slice reallocates and
relocates its elements, which trips that guard or silently churns widget
identities (losing each row's focus, scroll position, and animation state).
`WidgetSlice[*T]` keeps every element at a stable address across `SetLen`, so
identity survives a resize.

Give the row a `Measure` returning its intrinsic height so the parent layout can
size it; lay rows out with `FixedSize(rowHeight)` items in a vertical
`LinearLayout`. (`basicwidget.List[T]` / `Table[T]` handle scrolling lists for
you — prefer them for large collections.)

## State changes and when the screen updates

This is the second thing people get wrong. A rebuild/redraw happens
automatically when:

- an **input handler or event handler runs** (the framework assumes it may have
  mutated state), or
- a widget's **state key changes** (see `WriteStateKey`), or
- you explicitly call `guigui.RequestRebuild(widget)`.

So a counter mutated inside a button's `OnUp` updates with no extra work. But a
field mutated **outside** any handler — from a `Tick`, a goroutine result
applied on the main goroutine, or an ancestor reacting to an external model —
will *not* repaint unless you do one of:

1. **Implement `WriteStateKey`** to hash the state that should drive rebuilds.
   The framework re-hashes after each build and rebuilds when the hash changes:

   ```go
   func (w *Foo) WriteStateKey(s *guigui.StateKeyWriter) {
       s.WriteInt(w.count)
       s.WriteBool(w.open)
       s.WriteString(w.title)
   }
   ```

   `StateKeyWriter` has `WriteBool`, `WriteInt`/`WriteUint` (and sized
   variants), `WriteFloat32/64`, `WriteString`, `WriteWidget`, and raw `Write`.
   Writing nothing opts out (the default).

2. **Call `guigui.RequestRebuild(widget)`** at the mutation site if you do not
   want to maintain a state key.

`guigui.RequestRedraw(widget)` forces a repaint **without** a rebuild — use it
only when nothing in the tree structure changed (e.g. an animation frame).

## Context utilities

`*guigui.Context` (passed to most methods) also exposes per-widget state setters,
all called from `Build`:

- `context.SetEnabled(w, bool)` / `IsEnabled(w)`
- `context.SetVisible(w, bool)` / `IsVisible(w)`
- `context.SetFocused(w, bool)` / `IsFocused(w)`
- `context.SetOpacity(w, float64)`

## Handling input directly

Override `HandlePointingInput` / `HandleButtonInput` and return a result:

```go
func (w *Foo) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if widgetBounds.IsHitAtCursor() && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		// react...
		return guigui.HandleInputByWidget(w) // consume; ancestors won't see it
	}
	return guigui.HandleInputResult{} // not handled; let others try
}
```

Pointer/button input is delivered **post-order** (innermost/topmost widget
first), so a child can consume an event before its ancestors. Return
`guigui.HandleInputByWidget(w)` to consume, `guigui.AbortHandlingInputByWidget(w)`
to stop propagation without claiming the event, or the zero
`guigui.HandleInputResult{}` to pass. `widgetBounds.IsHitAtCursor()` reports
whether this widget is the topmost one under the cursor.

## Checklist when adding a widget

1. Embed `guigui.DefaultWidget`; keep children as plain fields; ensure the zero
   value is usable.
2. In `Build`: first add the children you use this rebuild with
   `adder.AddWidget(&w.child)` (skip ones that are not shown, like a closed
   popup's contents), then configure children from current state (idempotently).
   Register `On…` handlers here. Configuring a child you did not add is fine.
3. In `Layout`: position children with a `LinearLayout` (reuse the items slice)
   or `layouter.LayoutWidget`. Size in `UnitSize` multiples.
4. Add `Measure` if the widget has an intrinsic size (especially list rows). For
   a composite, share one `layout()` helper between `Layout` and `Measure`.
5. Read shared models via `context.Env`; bubble events up with a generated
   `EventKey` + `On…` setter + `DispatchEvent`.
6. If a field can change outside input/event handlers, expose it via
   `WriteStateKey` or call `RequestRebuild` when you mutate it.

## Verify your work — do not trust this file alone

This skill is a primer; it does not enumerate every widget method and it can
drift from an alpha API. Before considering a change done:

- **Look up real signatures, don't guess.** When you need a `basicwidget`
  method (`SetText`, `OnUp`, a `Set*`/`On*` you half-remember), grep the actual
  package: `grep -rn "func (.*Button)" basicwidget/`. The catalog here is
  intentionally not exhaustive.
- **Copy from a working example.** `example/counter` (state + buttons),
  `example/todo` (Env, events, dynamic list), and `example/gallery` (most
  widgets) are canonical, compiling usage. Prefer adapting them to inventing.
- **Build and vet.** `go build ./...` and `go vet ./...`. Guigui code that
  misuses the lifecycle often still compiles, so also run the program (or the
  relevant `example/`) and confirm it renders and reacts.
- **Sanity-check the lifecycle, not just the compile.** If a change does not
  show up, re-read "State changes and when the screen updates" — a clean build
  with a stale screen is the signature of a missing rebuild trigger.
- **A reduced repro passing is not proof the real app works.** A widget hosted
  in an isolated debug harness can render and behave correctly while the full
  application still fails: popups, overlay layers, the surrounding shell, and
  real pointer-input timing all differ. Confirm a UI fix in the actual app, not
  only in a harness.

## Common pitfalls

- **Reading bounds in `Build`.** Bounds are unknown there. Move it to `Layout`.
- **Configuring content in `Layout`.** Setters belong in `Build`; `Layout` only
  positions.
- **Registering handlers once / outside `Build`.** Handlers are wiped each
  rebuild — register them in `Build` every time.
- **Expecting a field write to repaint.** Only handler-driven or
  state-key-driven changes auto-rebuild; otherwise call `RequestRebuild`.
- **Allocating a fresh items slice every `Layout`.** Reuse with
  `slices.Delete(s, 0, len(s))`; `Layout` runs frequently.
- **Hard-coded pixel sizes.** Use `basicwidget.UnitSize(context)` so layouts
  scale with DPI and app scale.
- **Holding children in a plain value slice.** `append` to a `[]Row` reallocates
  and moves its elements; a widget's identity is the address of its embedded
  `DefaultWidget`, and moving/copying one by value panics (*"illegal use of
  DefaultWidget copied by value"*). Use `guigui.WidgetSlice[*T]` for a variable
  number of children (see "Dynamic lists of children").
- **Trying to scroll a panel past its content.** `basicwidget.Panel` clamps the
  scroll offset to `[min(viewport − content, 0), 0]`: positive offsets are
  discarded, and when the content fits the viewport the offset stays pinned at
  the origin. `SetScrollOffset` / `ForceSetScrollOffset` cannot push content to a
  positive offset — so e.g. centering only takes effect on an axis that actually
  overflows.
