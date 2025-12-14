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

	guigui.RegisterEventHandler2(r, &r.textInput)

	r.createButton.SetText("Create")
	guigui.RegisterEventHandler2(r, &r.createButton)
	context.SetEnabled(&r.createButton, r.model.CanAddTask(r.textInput.Value()))

	guigui.RegisterEventHandler2(r, &r.tasksPanelContent)
	r.tasksPanel.SetContent(&r.tasksPanelContent)
	r.tasksPanel.SetAutoBorder(true)
	r.tasksPanel.SetContentConstraints(basicwidget.PanelContentConstraintsFixedWidth)

	return nil
}

func (r *Root) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	switch targetWidget {
	case &r.textInput:
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.TextInputEventArgsKeyJustPressed:
			if eventArgs.Key == ebiten.KeyEnter {
				r.tryCreateTask(r.textInput.Value())
			}
		}
	case &r.createButton:
		switch eventArgs.(type) {
		case *basicwidget.ButtonEventArgsUp:
			r.tryCreateTask(r.textInput.Value())
		}
	case &r.tasksPanelContent:
		switch eventArgs := eventArgs.(type) {
		case *tasksPanelContentEventArgsDeleted:
			r.model.DeleteTaskByID(eventArgs.ID)
		}
	}
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

type taskWidgetEventArgsDoneButtonPressed struct{}

type taskWidget struct {
	guigui.DefaultWidget

	doneButton basicwidget.Button
	text       basicwidget.Text
}

func (t *taskWidget) SetText(text string) {
	t.text.SetValue(text)
}

func (t *taskWidget) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&t.doneButton)
	adder.AddChild(&t.text)

	t.doneButton.SetText("Done")
	guigui.RegisterEventHandler2(t, &t.doneButton)

	t.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	return nil
}

func (t *taskWidget) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	switch targetWidget {
	case &t.doneButton:
		switch eventArgs.(type) {
		case *basicwidget.ButtonEventArgsUp:
			guigui.DispatchEventHandler2(t, &taskWidgetEventArgsDoneButtonPressed{})
		}
	}
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

type tasksPanelContentEventArgsDeleted struct {
	ID int
}

type tasksPanelContent struct {
	guigui.DefaultWidget

	taskWidgets []taskWidget
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
		guigui.RegisterEventHandler2(t, &t.taskWidgets[i])
		t.taskWidgets[i].SetText(task.Text)
	}
	return nil
}

func (t *tasksPanelContent) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	idx := -1
	for i := range t.taskWidgets {
		if targetWidget == &t.taskWidgets[i] {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}

	model := context.Model(t, modelKeyModel).(*Model)
	switch eventArgs.(type) {
	case *taskWidgetEventArgsDoneButtonPressed:
		guigui.DispatchEventHandler2(t, &tasksPanelContentEventArgsDeleted{
			ID: model.TaskByIndex(idx).ID,
		})
	}
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
