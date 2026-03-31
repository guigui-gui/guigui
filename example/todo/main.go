// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	_ "github.com/guigui-gui/guigui/basicwidget/cjkfont"
)

var (
	modelKeyModel = guigui.GenerateEnvKey()
)

type Root struct {
	guigui.DefaultWidget

	background        basicwidget.Background
	createButton      basicwidget.Button
	textInput         basicwidget.TextInput
	tasksPanel        basicwidget.Panel
	tasksPanelContent tasksPanelContent

	model Model

	inputRowLayout guigui.LinearLayout
	inputRowItems  []guigui.LinearLayoutItem
	layoutItems    []guigui.LinearLayoutItem
}

func (r *Root) Env(context *guigui.Context, key guigui.EnvKey, source *guigui.EnvSource) (any, bool) {
	switch key {
	case modelKeyModel:
		return &r.model, true
	default:
		return nil, false
	}
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.textInput)
	adder.AddWidget(&r.createButton)
	adder.AddWidget(&r.tasksPanel)

	r.textInput.OnHandleButtonInput(func(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			r.tryCreateTask(r.textInput.Value())
			return guigui.HandleInputByWidget(&r.textInput)
		}
		return guigui.HandleInputResult{}
	})

	r.createButton.SetText("Create")
	r.createButton.OnUp(func(context *guigui.Context) {
		r.tryCreateTask(r.textInput.Value())
	})
	context.SetEnabled(&r.createButton, r.model.CanAddTask(r.textInput.Value()))

	r.tasksPanelContent.OnDeleted(func(context *guigui.Context, id int) {
		r.model.DeleteTaskByID(id)
	})
	r.tasksPanel.SetContent(&r.tasksPanelContent)
	r.tasksPanel.SetAutoBorder(true)
	r.tasksPanel.SetContentConstraints(basicwidget.PanelContentConstraintsFixedWidth)

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	r.inputRowItems = slices.Delete(r.inputRowItems, 0, len(r.inputRowItems))
	r.inputRowItems = append(r.inputRowItems,
		guigui.LinearLayoutItem{
			Widget: &r.textInput,
			Size:   guigui.FlexibleSize(1),
		},
		guigui.LinearLayoutItem{
			Widget: &r.createButton,
			Size:   guigui.FixedSize(5 * u),
		},
	)
	r.inputRowLayout = guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     r.inputRowItems,
		Gap:       u / 2,
	}
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Size:   guigui.FixedSize(u),
			Layout: &r.inputRowLayout,
		},
		guigui.LinearLayoutItem{
			Widget: &r.tasksPanel,
			Size:   guigui.FlexibleSize(1),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Gap:       u / 2,
		Padding: guigui.Padding{
			Start:  u / 2,
			Top:    u / 2,
			End:    u / 2,
			Bottom: u / 2,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (r *Root) tryCreateTask(text string) {
	if r.model.TryAddTask(text) {
		r.textInput.ForceSetValue("")
	}
}

type taskWidget struct {
	guigui.DefaultWidget

	doneButton basicwidget.Button
	text       basicwidget.Text

	layoutItems []guigui.LinearLayoutItem
}

var (
	taskWidgetEventDoneButtonPressed guigui.EventKey = guigui.GenerateEventKey()
)

func (t *taskWidget) OnDoneButtonPressed(f func(context *guigui.Context)) {
	guigui.SetEventHandler(t, taskWidgetEventDoneButtonPressed, f)
}

func (t *taskWidget) SetText(text string) {
	t.text.SetValue(text)
}

func (t *taskWidget) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&t.doneButton)
	adder.AddWidget(&t.text)

	t.doneButton.SetText("Done")
	t.doneButton.OnUp(func(context *guigui.Context) {
		guigui.DispatchEvent(t, taskWidgetEventDoneButtonPressed)
	})

	t.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	return nil
}

func (t *taskWidget) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	t.layoutItems = slices.Delete(t.layoutItems, 0, len(t.layoutItems))
	t.layoutItems = append(t.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &t.doneButton,
			Size:   guigui.FixedSize(3 * u),
		},
		guigui.LinearLayoutItem{
			Widget: &t.text,
			Size:   guigui.FlexibleSize(1),
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items:     t.layoutItems,
		Gap:       u / 2,
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *taskWidget) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return image.Pt(6*int(basicwidget.UnitSize(context)), t.doneButton.Measure(context, guigui.Constraints{}).Y)
}

type tasksPanelContent struct {
	guigui.DefaultWidget

	taskWidgets guigui.WidgetSlice[*taskWidget]

	layoutItems []guigui.LinearLayoutItem
}

var (
	tasksPanelContentEventDeleted guigui.EventKey = guigui.GenerateEventKey()
)

func (t *tasksPanelContent) OnDeleted(f func(context *guigui.Context, id int)) {
	guigui.SetEventHandler(t, tasksPanelContentEventDeleted, f)
}

func (t *tasksPanelContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	v, ok := context.Env(t, modelKeyModel)
	if !ok {
		return nil
	}
	model := v.(*Model)

	t.taskWidgets.SetLen(model.TaskCount())
	for i := range t.taskWidgets.Len() {
		adder.AddWidget(t.taskWidgets.At(i))
	}

	for i := range model.TaskCount() {
		task := model.TaskByIndex(i)
		t.taskWidgets.At(i).OnDoneButtonPressed(func(context *guigui.Context) {
			guigui.DispatchEvent(t, tasksPanelContentEventDeleted, task.ID)
		})
		t.taskWidgets.At(i).SetText(task.Text)
	}
	return nil
}

func (t *tasksPanelContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	t.layoutItems = slices.Delete(t.layoutItems, 0, len(t.layoutItems))
	for i := range t.taskWidgets.Len() {
		w := widgetBounds.Bounds().Dx()
		h := t.taskWidgets.At(i).Measure(context, guigui.FixedWidthConstraints(w)).Y
		t.layoutItems = append(t.layoutItems, guigui.LinearLayoutItem{
			Widget: t.taskWidgets.At(i),
			Size:   guigui.FixedSize(h),
		})
	}
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     t.layoutItems,
		Gap:       u / 4,
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *tasksPanelContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := basicwidget.UnitSize(context)
	var h int
	for i := range t.taskWidgets.Len() {
		h += t.taskWidgets.At(i).Measure(context, constraints).Y
		h += int(u / 4)
	}
	w := t.DefaultWidget.Measure(context, constraints).X
	return image.Pt(w, h)
}

func main() {
	op := &guigui.RunOptions{
		Title:         "TODO",
		WindowMinSize: image.Pt(320, 240),
		RunGameOptions: &ebiten.RunGameOptions{
			ApplePressAndHoldEnabled: true,
		},
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
