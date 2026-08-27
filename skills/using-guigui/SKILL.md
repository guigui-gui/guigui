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

> **Freshness.** Verified against Guigui at commit `c1ffb301` (2026-07-10).
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
  at the app's TPS — 60×/sec by default, or whatever TPS is configured) first
  settles the tree: it may run **Build** (reconstruct the child tree),
  **Layout** (position the children), and the **`Handle*Input`** methods before
  calling **Tick** exactly once. A state change during input can cause another
  Build/Layout pass in the same tick before `Tick` runs. Rely only on these
  guarantees: Build precedes Layout within a pass, input sees the current
  layout, and Tick runs once, after input handling. Do not assume a fixed count
  of Build or Layout passes per tick. Each **frame** (Ebitengine's `Draw`, at
  the display's refresh rate) does **Draw**. Ticks and frames are *not*
  one-to-one — there may be more or fewer frames than ticks — so put per-step
  logic in `Tick`, never in `Draw`.
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
`WindowSize` / `WindowMinSize` / `WindowMaxSize`, `WindowFloating`, `AppScale`,
and an optional `RunGameOptions` passed through to Ebitengine.

## The Widget interface

Embed `guigui.DefaultWidget`, then override as needed:

| Method | Signature | Override when |
|---|---|---|
| `Build` | `Build(*Context, *ChildAdder) error` | Almost always — declare and configure children. |
| `Layout` | `Layout(*Context, *WidgetBounds, *ChildLayouter)` | You have children to position (i.e. almost always). |
| `Measure` | `Measure(*Context, Constraints) image.Point` | The widget has an intrinsic preferred size (list rows, leaf widgets). |
| `HandlePointingInput` | `HandlePointingInput(*Context, *WidgetBounds) HandleInputResult` | Custom mouse/touch handling. |
| `HandleButtonInput` | `HandleButtonInput(*Context, *WidgetBounds) HandleInputResult` | Custom keyboard/gamepad handling — only delivered under focus/receptiveness conditions (see "Handling input directly"). |
| `Env` | `Env(*Context, EnvKey, *EnvSource) (any, bool)` | The widget provides shared values to descendants. |
| `WriteStateKey` | `WriteStateKey(*Context, *StateKeyWriter)` | State changes outside input/events must trigger a rebuild (see below). |
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

`widgetBounds.VisibleBounds()` gives the part left after ancestor clipping.
Use `Bounds()` for layout coordinates and `VisibleBounds()` to cull custom
drawing. For ordinary cursor hit testing, use `widgetBounds.IsHitAtCursor()`;
it accounts for clipping and higher-layer occlusion. Use `VisibleBounds()`
geometrically only for an additional custom-shape check or a point other than
the cursor. In `Draw`, `dst` is a clipped subimage whose coordinates are still
screen coordinates, so draw at `Bounds().Min` (or other screen-space
coordinates), not at an assumed `(0, 0)` origin.

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
r.createButton.OnDown(func(context *guigui.Context) {
	r.addItem(r.textInput.Value())
})
context.SetEnabled(&r.createButton, r.canAdd())
```

For button actions, default to `OnDown`; reach for `OnUp` only when release
semantics genuinely matter.

Use `guigui.WidgetWithSize[*T]` or `guigui.WidgetWithPadding[*T]` when a child
only needs an intrinsic-size override or padding. These wrappers lazily create
their zero-value child, forward layout and focus, and expose it through
`Widget()`. Use `guigui.LayerWidget[*T]` for floating content that must escape
ancestor clipping or sit above other input targets: add and lay out the wrapper,
then call `BringToFrontLayer(context)` during `Build`. Higher layers draw and
receive input first.

A widget instance can have only one parent in a build. Do not reuse the same
content widget in two lists, a source view and a popup, or any other two places;
create distinct widget instances that render the same data. Sharing the model is
fine, but sharing the widget corrupts its parent chain, env lookup, bounds, and
focus/input behavior.

### Keep in-progress edits separate from model synchronization

For an editable `basicwidget.TextInput`, call `SetValue(modelValue)` during
`Build` as usual. It preserves an active user edit instead of overwriting the
live buffer on every whole-tree rebuild. Reserve `ForceSetValue` for an explicit
replacement/reset: it immediately replaces the buffer and can synchronously
invoke `OnValueChanged` as a committed change. If a forced seed must not be
treated as user input, guard the callback with a short-lived `seeding` flag.

When replacing a subtree or switching records, commit or deliberately discard
active edits before resetting the widget. Otherwise the model and the retained
editing buffer can silently diverge.

### Custom list item content must color itself

`basicwidget.ListItem[T].Content` takes any widget, which is how a row gets an
icon, several columns, or an editable field. Setting `Content` replaces the
item's built-in text widget, so the list stops coloring anything inside the row.
It still paints the selection highlight behind the row, so a custom row that
ignores its state keeps its default text color and turns unreadable on the
accent background.

The list publishes each row's state as a `basicwidget.ListItemColorType` under
`basicwidget.EnvKeyListItemColorType` (`Default`, `Highlighted`,
`SelectedInUnfocusedList`, `Hovered`, `ItemDisabled`, `ListDisabled`). Read it
and apply `TextColor(context)`:

```go
func (r *row) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	colorType := basicwidget.ListItemColorTypeDefault
	if v, ok := context.Env(r, basicwidget.EnvKeyListItemColorType); ok {
		if ct, ok := v.(basicwidget.ListItemColorType); ok {
			colorType = ct
		}
	}
	var style basicwidget.TextStyle
	style.SetColor(colorType.TextColor(context))
	r.text.SetBaseStyle(&style)

	r.layout(context).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}
```

What that shape encodes:

- **Read it in `Layout` (or `Draw`), not `Build`.** This is the deliberate
  exception to "setters belong in `Build`": the value comes from the list's own
  content widget, so it exists only after the build phase.
- **Guard the lookup *and* the type assertion.** A missing value is normal, not
  an error: the lookup returns `(nil, false)` for a row scrolled out of the
  viewport and for the same widget used outside any list. `basicwidget.Popup`
  additionally returns `(nil, true)` to stop a row's color type from leaking
  into popup content, so `ok` alone does not mean you got a
  `ListItemColorType` — always use the comma-ok form, never a bare
  `v.(basicwidget.ListItemColorType)`.
- **Falling back to `Default` is the simplest correct choice**; it maps to the
  ordinary text color. Leaving the color `nil` and not touching the style also
  works, and suits a widget that has its own palette when used outside a list.
- **The color type drives more than text.** `BackgroundColor(context)` returns
  the matching fill (`nil` for `Default`), and `Draw` can branch on
  `Highlighted` to swap a border or fill. A monochrome icon needs re-rendering
  with `ebiten.ColorModeDark` on a highlighted row, because the accent
  background wants a light foreground.

Use `basicwidget.ListItemTextPadding(context)` as the row's padding so custom
content lines up with ordinary text items. `example/gallery/selects.go`
(`selectItem`: icon + text) and `example/biglist` (`itemWidget`: variable row
heights) are the in-tree implementations to copy.

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
presentation state down to descendant widgets. A `List` advertises each row's
state through `basicwidget.EnvKeyListItemColorType` so that custom row content
can color itself (see "Custom list item content must color itself"). So when
composing a custom widget that lives inside a built-in container, check whether
that container exposes an `EnvKey…` you should honor.

### Env lookups on a just-created widget silently find nothing

`context.Env(widget, key)` resolves by walking up `widget`'s parent pointers,
and a widget only gets its parent pointer when some ancestor's `Build` adds it
with `adder.AddWidget`. A widget that no `Build` has added yet has no parent, so
the lookup returns `(nil, false)` — no panic, no error, just a missing value.

Two consequences:

- **On yourself, inside your own `Build` / `Layout`, env lookups are safe.** By
  the time your `Build` runs, the parent's `Build` has already added you, so
  your parent chain is intact.
- **A parent's `Build` must not resolve env *through* a child that may have
  just been created.** The classic sequence: an add-item handler grows a
  `WidgetSlice` pool and selects the new element. On the very next `Build` of
  the owner, that element is still unparented — it only gains a parent later in
  the same build pass, when the container hosting it builds. Any env lookup
  routed through it fails, and code assuming a non-nil result panics.

```go
// An event handler grows the pool and selects the new element:
w.addButton.OnDown(func(context *guigui.Context) {
	w.rowContents.SetLen(w.rowContents.Len() + 1)
	w.selectedRow = w.rowContents.Len() - 1
})

// BAD — the owner's next Build resolves env through the fresh element:
content := w.rowContents.At(w.selectedRow)
v, _ := context.Env(content, envKeyModel) // fresh element: parent is nil → (nil, false)
model, _ := v.(*Model)                    // model is nil → panics on first use
```

Resolve shared state through the widget whose `Build` is running — its own env
is always available there — and hand data to children via their plain fields:

```go
// GOOD — inside (w *Editor) Build:
v, ok := context.Env(w, envKeyModel)
if !ok {
	return nil
}
model := v.(*Model)
// ... configure w.rowContents elements from model via their Set* methods ...
```

If a widget keeps a helper that resolves a model via its own env (e.g.
`func (c *Content) model(context) *Model` built on `context.Env(c, …)`), make it
return nil when the lookup fails and guard at every call site — the lookup can
legitimately fail while the widget is not (yet) in the tree.

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
	w.deleteButton.OnDown(func(context *guigui.Context) {
		guigui.DispatchEvent(w, eventItemDeleted, w.id)
	})
	return nil
}
```

The parent calls `child.OnDeleted(func(context, id){ ... })` during its own
`Build`. Handlers are **cleared and re-registered on every rebuild**, so always
(re)register inside `Build`. This is exactly how `basicwidget` buttons expose
`OnDown` and `OnUp`.

Re-registering a handler does not require allocating a new closure each time.
In a large tree, keep hot handlers in widget fields, initialize them once when
nil, and pass the same function back to `On...` during every `Build`. A cached
closure should capture only the stable widget receiver (or another stable key):
never retain the `Context` argument or a Build-local model/item whose meaning can
change. Use the context passed to the callback and resolve current data from the
receiver or `Env` when the event actually runs.

## Dynamic lists of children

For a variable number of children, use `guigui.WidgetSlice[*T]`: set its length
to match the data, then add and configure each element.

```go
func (l *List) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	n := l.model.Count()
	l.rows.SetLen(n)
	for i := range n {
		adder.AddWidget(l.rows.At(i))
	}
	for i := range n {
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

`WidgetSlice` preserves widget addresses, not the logical identity of the data
shown in each slot. For filterable or reorderable data, track selection and
per-item state by a stable item value/key, reconfigure every slot from current
data in `Build`, and validate the selection after deletion. A row index can
start referring to a different item after a rebuild.

## Resetting a widget's cached state

A widget accumulates state that is *not* re-derived from its inputs each Build —
a selected index, a scroll offset, a cache keyed by IDs, an in-progress edit.
When the data behind it is swapped wholesale (a document reloaded, a different
record opened), that state points at the old data and won't self-heal. Drop it by
assigning the widget its zero value, at the top of **`Build`**, before re-adding
the child:

```go
func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	if doc := r.model.Document(); doc != r.lastDoc {
		r.lastDoc = doc     // a fresh object means the data was replaced
		r.editor = Editor{} // discard the child's accumulated state
	}
	// ... add and configure children ...
}
```

This is safe as far as the framework is concerned: every widget must be usable
from its zero value, so the field keeps its address, `copyCheck` re-pins, identity
holds, and the next Build re-adds and reconfigures the fresh child. It drops only
in-memory state, though — a widget that owns an external resource (a live
connection, a goroutine, an open file) must release it *before* zeroing, since
Guigui has no teardown hook and a bare reset would leak it. Key the trigger on
object identity (or a generation counter), not a name/path, so a reload that
yields a fresh object at the same path still fires.

Reset in `Build`, not `Tick`: `Tick` runs after that tick's Layout, so a tree
change there is inconsistent with the frame's already-computed layout for one
Draw. And never zero the **application root** widget's own struct this way — the
framework marks the root once at startup and never re-establishes it, so wiping
its embedded `DefaultWidget` unmarks it. Reset the root's *children*, not the
root value itself.

## State changes and when the screen updates

This is the second thing people get wrong. Tree/content changes need a rebuild;
paint-only changes need only a redraw.

Rebuilds happen when:

- an **input handler or event handler runs** (the framework assumes it may have
  mutated state),
- a widget's **state key changes** (see `WriteStateKey`), or
- you explicitly call `guigui.RequestRebuild()`.

**A rebuild is whole-tree, never scoped.** Any of these re-runs `Build` on
*every* widget from the root — there is no per-widget or per-subtree rebuild.
`RequestRebuild()` takes no argument; a changed state key likewise rebuilds the
whole tree, not just the widget that owns the key. So place a `WriteStateKey`
on whichever widget *owns* the self-changing state — a **different** widget
that renders from it still re-runs its own `Build` and reflects the change, so
the key need not live on the widget that displays it.

**A rebuild re-runs `Build`; it does not repaint by itself.** The pixels that
get redrawn come from state-key changes (which auto-repaint the changed
widget's region), from tree/layout diffs, or from an explicit `RequestRedraw`.
A bare `RequestRebuild()` seeds no redraw region, so for a paint-only change
that alters a widget's appearance without changing its bounds or a state key,
pair it with `guigui.RequestRedraw(widget)`.

So a counter mutated inside a button's `OnUp` updates with no extra work. But a
field mutated **outside** any handler — from a `Tick`, a goroutine result
applied on the main goroutine, or an ancestor reacting to an external model —
will *not* repaint unless you do one of:

1. **Implement `WriteStateKey`** to hash the state that should drive rebuilds.
   The framework re-hashes after each build and rebuilds when the hash changes:

   ```go
   func (w *Foo) WriteStateKey(context *guigui.Context, s *guigui.StateKeyWriter) {
       s.WriteInt(w.count)
       s.WriteBool(w.open)
       s.WriteString(w.title)
   }
   ```

   `StateKeyWriter` has `WriteBool`, `WriteInt`/`WriteUint` (and sized
   variants), `WriteFloat32/64`, `WriteString`, `WriteWidget`, and raw `Write`.
   Writing nothing opts out (the default).

2. **Call `guigui.RequestRebuild()`** at the mutation site when the key
   mechanism structurally cannot observe the change. Keys are only checked for
   widgets currently in the tree, so this is needed when:

   - the owning widget may be **out of the tree** at mutation time — a widget
     gated out of its parent's `Build` has no key evaluated, so a transition
     that should bring it back in (e.g. reopening a closed popup) must be an
     explicit rebuild;
   - the mutation has **no owner widget** — app-level code mutates a shared
     model that no widget hashes (though giving the root, or whichever widget
     owns the model, a key or generation counter is usually the better fix);
   - the state is **too expensive to hash** at every checkpoint and no cheap
     generation counter is feasible.

   `RequestRebuild` re-runs `Build` but does not repaint on its own; pair it
   with `guigui.RequestRedraw(widget)` when the appearance also changes within
   unchanged bounds. For a purely paint-only change, skip the rebuild and call
   `RequestRedraw` alone.

`guigui.RequestRedraw(widget)` forces a repaint **without** a rebuild — use it
only when nothing in the tree structure changed (e.g. an animation frame).

**Choosing between them: prefer `WriteStateKey` in usual cases.** The
exception is paint-only state. For state mutated outside `Build`, ask: does
the change alter what `Build` or `Measure` would produce? If yes (a child
appears or disappears, a size animates), put it in `WriteStateKey` — the
rebuild a key change triggers is exactly what's needed. If no — the widget
merely paints differently inside unchanged bounds (hover or pressed tinting,
an alpha fade, a caret blink, a thumb sliding within the widget) — keep that
state **out** of the key and call `RequestRedraw` at the mutation site. A
state-key change rebuilds the whole tree, so for a paint-only change every
`Build` would re-run only to produce the identical tree, at whole-tree cost
per animation tick or hover flip.

Caveat: paint-only state that is only ever set from an ancestor's `Build`
(e.g. a flag a parent pushes down on each build) is still fine in the key —
that path is detected by the post-build snapshot and repaints without any
extra rebuild. The rebuild cost bites only for state that changes at
`Tick`/input-handling time.

A state key rebuilds only when its hashed bytes *change*, so a value that holds
steady across ticks (e.g. an `isPlaying` flag that stays `true` for the whole
animation) costs nothing per tick. Pair the two: drive the per-frame animation
with `RequestRedraw` each tick, and let a `WriteStateKey` fire the one rebuild
on the on/off transition (when the animation finishes and a button must flip
from stop back to play).

For a large mutable model, prefer hashing a cheap monotonic generation counter
in `WriteStateKey` over serializing every collection. Increment the generation
for every mutation that affects the UI. State deliberately omitted from the key
still needs an explicit `RequestRebuild` at its mutation sites.

The ladder, in order: `WriteStateKey` by default; `RequestRedraw` for
paint-only state; `RequestRebuild` only when the key mechanism structurally
cannot observe the change.

### A keyless widget drawing state its parent's Build hands it

The subtlest gap in the ladder, and the inverse of the caveat above: a widget
with **no** `WriteStateKey` whose own `Draw` renders fields that a parent's
`Build` pushes into it via setters each rebuild.

The rebuild side works fine — the parent re-runs, calls the setters, the
widget holds the new values. But a rebuild repaints nothing by itself, and the
post-build snapshot that turns "this widget's state changed during the build"
into a repaint of its region works by comparing `WriteStateKey` hashes. A
widget that writes no key always hashes the same, so the framework never
notices, and the screen keeps the old pixels while the struct holds the new
state. Children configured by that widget's `Build` are unaffected (their own
keys change), which makes the failure look absurd: the `Text` next to the
thumbnail updates while the thumbnail itself does not.

The symptom is a stale region that heals on its own. A focus change repaints
the whole screen, so clicking anywhere that moves focus fixes it; only
focus-preserving mutations leave it visible — arrow-key or key-repeat value
changes in a focused input, undo/redo shortcuts, results applied from a
goroutine. One stale region with correct neighbors, fixed by the next click,
is this bug's signature.

The fix is a `WriteStateKey` hashing exactly what `Draw` reads. When `Draw`
also renders shared data reached via `Env` (a document, a map, a model), hash
that model's generation counter too, so edits to it repaint the widget as
well.

### Auditing a widget for missing key coverage

When reviewing a widget (or hunting a stale-screen bug), the defect pattern
is: a setter stores a field on the widget; the field reaches the screen
through the widget's own `Build`/`Layout`/`Measure`/`Draw` (directly or via
helpers they call); and the field is neither written into `WriteStateKey` nor
accompanied by `RequestRebuild`/`RequestRedraw` at the mutation site. There
are two flavors: a `Build`-read field misses its **rebuild** when set outside
a build pass, while a `Draw`-read field misses its **repaint** even when set
during one (previous section).

Before flagging one, rule out what is safe by construction:

- **The setter forwards to a child widget's own setter** (`w.text.SetValue(v)`)
  instead of storing the value — the child's key covers it.
- **The mutation happens in a `DispatchEvent` handler or in an input handler
  that returns `HandleInputByWidget`** — both force a whole-tree rebuild, which
  is why most callback-driven state needs no key. (Handlers that mutate and
  return an empty `HandleInputResult{}` get no such rebuild.)
- **The field is read only inside `On…` callbacks** — that is behavior, not
  screen state; whatever the callback changes triggers its own update.
- **The field is read only in `CursorShape`** — the cursor shape is
  re-evaluated every frame without any rebuild.
- **The value is derived from `Env` or other widgets' state during `Build`** —
  it can only change during a build pass anyway.

One clearing argument deserves suspicion: "safe because every caller also
flips something else that rebuilds at the same time" — sets a field right
before opening a popup (`Popup.SetOpen` requests a rebuild itself), or only
ever together with a sibling field that *is* in the key. Such coupling works
but is invisible at the widget itself and breaks silently when a future call
site changes the setter's company. When you find yourself relying on it,
prefer adding the field to the key — hashing one more int is cheap insurance.

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

`HandleButtonInput` is not delivered to every widget. It runs only when the
widget is focused, has a focused ancestor or descendant, or is itself marked
button-input-receptive with `context.SetButtonInputReceptive(w, true)`. Focus
lights up a whole path — the focused widget, its ancestors, and its descendants
all receive button input. Receptiveness is strictly self-only: it grants
delivery to that one widget, not to its ancestors or descendants (the framework
traverses through ancestors only to reach the receptive widget). Overriding
`HandleButtonInput` on a widget that is never focused and never marked
receptive silently does nothing — focus the widget (`context.SetFocused`) or
mark it receptive.

Rich content nested inside a button or list item can intercept pointing input
before the interactive parent. If that content is decorative, call
`context.SetPassthrough(content, true)` in `Build`; the decorative subtree stops
receiving input, allowing the container to handle the click.

A visually modal popup does not automatically isolate keyboard shortcuts. Make
its content button-input-receptive, handle intended keys such as Escape, and
return `AbortHandlingInputByWidget` for the rest so input cannot leak to widgets
or global shortcut handlers behind it.

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
   `WriteStateKey` (preferred) or call `RequestRebuild` when you mutate it —
   unless the change is paint-only (unchanged bounds and children), in which
   case keep it out of the key and call `RequestRedraw` alone.
7. If the widget has its own `Draw`, put every field `Draw` reads into
   `WriteStateKey` even when only an ancestor's `Build` ever sets them — a
   keyless widget is rebuilt but never repainted (see "A keyless widget
   drawing state its parent's Build hands it").

## Verify your work — do not trust this file alone

This skill is a primer; it does not enumerate every widget method and it can
drift from an alpha API. Before considering a change done:

- **Look up real signatures, don't guess.** When you need a `basicwidget`
  method (`SetText`, `OnUp`, a `Set*`/`On*` you half-remember), search the actual
  package: `rg -n "func \\(.*Button\\)" basicwidget/`. The catalog here is
  intentionally not exhaustive.
- **Copy from a working example.** `example/counter` (state + buttons),
  `example/todo` (Env, events, dynamic list), `example/gallery` (most widgets),
  and `example/biglist` (custom list item content) are canonical, compiling
  usage. Prefer adapting them to inventing.
- **Build and vet.** `go build ./...` and `go vet ./...`. Guigui code that
  misuses the lifecycle often still compiles, so also run the program (or the
  relevant `example/`) and confirm it renders and reacts.
- **Prefer driving it headlessly.** A headless run is the better default even
  when a display is right there: it opens no window and never steals the
  keyboard focus, and the same script reruns identically. A Guigui app is an
  `ebiten.Game` — `guigui.Run` hands the root tree to
  `ebiten.RunGameWithOptions` — so the **`run-ebitengine-app-headless`** skill
  applies to it unchanged: it builds the app with `-tags ebitenginevm`, runs it
  as an `exp/vmhost` guest, steps ticks, injects pointer and keyboard input, and
  reads rendered frames back as pixels or PNGs. That skill lives in the
  Ebitengine repository (`skills/run-ebitengine-app-headless/`), so invoking it
  by name works only where it is installed. Use it for screenshots, golden-image
  diffs of a rendering change, and deterministic multi-tick repros.
  Guigui specifics to script around (`doc.go` documents every environment
  variable):
  - Pin what the host platform would otherwise decide, so two runs are
    comparable: `GUIGUI_COLOR_MODE` (`light`/`dark`),
    `GUIGUI_KEY_BINDING_MODE` (`command`/`control-default`/`control-emacs`),
    and `GUIGUI_DEBUG=devicescale=1`.
  - Injected runes do **not** reach a text input: text arrives through
    `exp/textinput` IME sessions, not `ebiten.AppendInputChars`. Seed text by
    pasting instead — `GUIGUI_DEBUG=emulateclipboard` plus
    `GUIGUI_DEBUG_CLIPBOARD_TEXT` gives the app an in-process clipboard.
  - Shortcut modifiers are read as the virtual `ebiten.KeyMeta`/`KeyControl`,
    which register only when a physical key such as `KeyMetaLeft` is injected.
  - `GUIGUI_DEBUG=showbuildlogs,showinputlogs` turns a silent "nothing happened"
    into a log of what rebuilt and which widget consumed the input.
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
- **Resetting a widget's cached state in `Tick`.** Zeroing a child to drop stale
  state belongs in `Build` (the rebuild boundary); `Tick` runs after `Layout`,
  so a tree change there is inconsistent with the frame's layout for one `Draw`.
  Never zero the application root's own struct (see "Resetting a widget's cached
  state").
- **Expecting a field write to repaint.** Only handler-driven or
  state-key-driven changes auto-rebuild; otherwise call `RequestRebuild` (or
  just `RequestRedraw` if the change is paint-only).
- **A custom `Draw` on a widget with no `WriteStateKey`.** Fields the parent's
  `Build` hands it arrive, but the region is never repainted — the post-build
  repaint compares key hashes, and a keyless widget always hashes the same.
  The staleness is intermittent because any focus change repaints the whole
  screen (see "A keyless widget drawing state its parent's Build hands it").
- **Allocating a fresh items slice every `Layout`.** Reuse with
  `slices.Delete(s, 0, len(s))`; `Layout` runs frequently.
- **Hard-coded pixel sizes.** Use `basicwidget.UnitSize(context)` so layouts
  scale with DPI and app scale.
- **Holding children in a plain value slice.** `append` to a `[]Row` moves its
  elements, which churns widget identity or panics. Use `guigui.WidgetSlice[*T]`
  for a variable number of children (see "Dynamic lists of children").
- **Reusing one widget in multiple containers.** A widget has one parent chain.
  Use distinct widget instances backed by shared data.
- **Forcing a text input's value on every Build.** `ForceSetValue` overwrites an
  active edit and can synchronously emit a committed change. Use `SetValue` for
  normal model synchronization and force only explicit resets.
- **Letting decorative content consume its container's click.** Mark rich
  button/list content passthrough so the interactive ancestor handles input.
- **Leaving custom list item content its default text color.** A
  `ListItem[T].Content` widget replaces the item's built-in text, so nothing
  recolors it and a highlighted row goes unreadable. Read
  `basicwidget.EnvKeyListItemColorType` in `Layout` (see "Custom list item
  content must color itself").
- **Resolving env through a widget that was just created.** A widget nobody has
  built yet has no parent, so the lookup silently returns `(nil, false)` and code
  assuming a non-nil value crashes. Resolve shared state through the widget whose
  `Build` is running, not through a possibly-new child (see "Env lookups on a
  just-created widget silently find nothing").
- **Overriding `HandleButtonInput` on a never-focused widget.** It reaches only
  focused widgets (plus their ancestors/descendants) or button-input-receptive
  ones. Focus the widget or call `context.SetButtonInputReceptive` (see
  "Handling input directly").
- **Trying to scroll a panel past its content.** `basicwidget.Panel` clamps the
  scroll offset to `[min(viewport − content, 0), 0]`: positive offsets are
  discarded, and when the content fits the viewport the offset stays pinned at
  the origin. `SetScrollOffset` / `ForceSetScrollOffset` cannot push content to a
  positive offset — so e.g. centering only takes effect on an axis that actually
  overflows.
