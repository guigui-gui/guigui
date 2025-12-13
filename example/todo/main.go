// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 The Guigui Authors

package main

import (
	"fmt"
	"image"
	"os"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
	"github.com/guigui-gui/guigui/basicwidget/cjkfont"
)

type modelKey int

const (
	modelKeyModel modelKey = iota
)

type Root struct {
	guigui.DefaultWidget

	background        basicwidget.Background
	createButton      basicwidget.Button
	textInput         basicwidget.TextInput
	tasksPanel        basicwidget.Panel
	tasksPanelContent tasksPanelContent

	model Model

	locales           []language.Tag
	faceSourceEntries []basicwidget.FaceSourceEntry
}

func (r *Root) updateFontFaceSources(context *guigui.Context) {
	r.locales = slices.Delete(r.locales, 0, len(r.locales))
	r.locales = context.AppendLocales(r.locales)

	r.faceSourceEntries = slices.Delete(r.faceSourceEntries, 0, len(r.faceSourceEntries))
	r.faceSourceEntries = cjkfont.AppendRecommendedFaceSourceEntries(r.faceSourceEntries, r.locales)
	basicwidget.SetFaceSources(r.faceSourceEntries)
}

func (r *Root) Model(key any) any {
	switch key {
	case modelKeyModel:
		return &r.model
	default:
		return nil
	}
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&r.background)
	adder.AddChild(&r.textInput)
	adder.AddChild(&r.createButton)
	adder.AddChild(&r.tasksPanel)

	r.updateFontFaceSources(context)

	r.textInput.SetOnKeyJustPressed(func(context *guigui.Context, key ebiten.Key) {
		if key == ebiten.KeyEnter {
			r.tryCreateTask(r.textInput.Value())
		}
	})

	r.createButton.SetText("Create")
	r.createButton.SetOnUp(func(context *guigui.Context) {
		r.tryCreateTask(r.textInput.Value())
	})
	context.SetEnabled(&r.createButton, r.model.CanAddTask(r.textInput.Value()))

	r.tasksPanelContent.SetOnDeleted(func(context *guigui.Context, id int) {
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
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Size: guigui.FixedSize(u),
				Layout: guigui.LinearLayout{
					Direction: guigui.LayoutDirectionHorizontal,
					Items: []guigui.LinearLayoutItem{
						{
							Widget: &r.textInput,
							Size:   guigui.FlexibleSize(1),
						},
						{
							Widget: &r.createButton,
							Size:   guigui.FixedSize(5 * u),
						},
					},
					Gap: u / 2,
				},
			},
			{
				Widget: &r.tasksPanel,
				Size:   guigui.FlexibleSize(1),
			},
		},
		Gap: u / 2,
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
}

const (
	taskWidgetEventDoneButtonPressed = "doneButtonPressed"
)

func (t *taskWidget) SetOnDoneButtonPressed(f func(context *guigui.Context)) {
	guigui.RegisterEventHandler(t, taskWidgetEventDoneButtonPressed, f)
}

func (t *taskWidget) SetText(text string) {
	t.text.SetValue(text)
}

func (t *taskWidget) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.doneButton)
	adder.AddChild(&t.text)

	t.doneButton.SetText("Done")
	t.doneButton.SetOnUp(func(context *guigui.Context) {
		guigui.DispatchEventHandler(t, taskWidgetEventDoneButtonPressed)
	})

	t.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	return nil
}

func (t *taskWidget) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionHorizontal,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &t.doneButton,
				Size:   guigui.FixedSize(3 * u),
			},
			{
				Widget: &t.text,
				Size:   guigui.FlexibleSize(1),
			},
		},
		Gap: u / 2,
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *taskWidget) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	return image.Pt(6*int(basicwidget.UnitSize(context)), t.doneButton.Measure(context, guigui.Constraints{}).Y)
}

type tasksPanelContent struct {
	guigui.DefaultWidget

	taskWidgets []taskWidget
}

const (
	tasksPanelContentEventDeleted = "deleted"
)

func (t *tasksPanelContent) SetOnDeleted(f func(context *guigui.Context, id int)) {
	guigui.RegisterEventHandler(t, tasksPanelContentEventDeleted, f)
}

func (t *tasksPanelContent) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	model := context.Model(t, modelKeyModel).(*Model)

	if model.TaskCount() > len(t.taskWidgets) {
		t.taskWidgets = slices.Grow(t.taskWidgets, model.TaskCount()-len(t.taskWidgets))[:model.TaskCount()]
	} else {
		t.taskWidgets = slices.Delete(t.taskWidgets, model.TaskCount(), len(t.taskWidgets))
	}
	for i := range t.taskWidgets {
		adder.AddChild(&t.taskWidgets[i])
	}

	for i := range model.TaskCount() {
		task := model.TaskByIndex(i)
		t.taskWidgets[i].SetOnDoneButtonPressed(func(context *guigui.Context) {
			guigui.DispatchEventHandler(t, tasksPanelContentEventDeleted, task.ID)
		})
		t.taskWidgets[i].SetText(task.Text)
	}
	return nil
}

func (t *tasksPanelContent) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	layout := guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Gap:       u / 4,
	}
	layout.Items = make([]guigui.LinearLayoutItem, len(t.taskWidgets))
	for i := range t.taskWidgets {
		w := widgetBounds.Bounds().Dx()
		h := t.taskWidgets[i].Measure(context, guigui.FixedWidthConstraints(w)).Y
		layout.Items[i] = guigui.LinearLayoutItem{
			Widget: &t.taskWidgets[i],
			Size:   guigui.FixedSize(h),
		}
	}
	layout.LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func (t *tasksPanelContent) Measure(context *guigui.Context, constraints guigui.Constraints) image.Point {
	u := basicwidget.UnitSize(context)
	var h int
	for i := range t.taskWidgets {
		h += t.taskWidgets[i].Measure(context, constraints).Y
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
