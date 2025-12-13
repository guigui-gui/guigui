// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Lists struct {
	guigui.DefaultWidget

	listFormPanel basicwidget.Panel
	listForm      basicwidget.Form
	listText      basicwidget.Text
	list          guigui.WidgetWithSize[*basicwidget.List[int]]
	treeText      basicwidget.Text
	tree          guigui.WidgetWithSize[*basicwidget.List[int]]

	jumpForm         basicwidget.Form
	indexText        basicwidget.Text
	indexNumberInput basicwidget.NumberInput
	jumpButton       basicwidget.Button

	configForm       basicwidget.Form
	showStripeText   basicwidget.Text
	showStripeToggle basicwidget.Toggle
	showHeaderText   basicwidget.Text
	showHeaderToggle basicwidget.Toggle
	showFooterText   basicwidget.Text
	showFooterToggle basicwidget.Toggle
	movableText      basicwidget.Text
	movableToggle    basicwidget.Toggle
	enabledText      basicwidget.Text
	enabledToggle    basicwidget.Toggle

	listItems []basicwidget.ListItem[int]
	treeItems []basicwidget.ListItem[int]
}

func (l *Lists) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddChild(&l.listFormPanel)
	adder.AddChild(&l.jumpForm)
	adder.AddChild(&l.configForm)

	model := context.Model(l, modelKeyModel).(*Model)

	u := basicwidget.UnitSize(context)

	// List
	l.listText.SetValue("Text list")

	list := l.list.Widget()
	list.SetStripeVisible(model.Lists().IsStripeVisible())
	if model.Lists().IsHeaderVisible() {
		list.SetHeaderHeight(u)
	} else {
		list.SetHeaderHeight(0)
	}
	if model.Lists().IsFooterVisible() {
		list.SetFooterHeight(u)
	} else {
		list.SetFooterHeight(0)
	}
	guigui.RegisterEventHandler2(l, list)

	l.listItems = slices.Delete(l.listItems, 0, len(l.listItems))
	l.listItems = model.Lists().AppendListItems(l.listItems)
	list.SetItems(l.listItems)
	context.SetEnabled(&l.list, model.Lists().Enabled())
	l.list.SetFixedHeight(6 * u)

	// Tree
	l.treeText.SetValue("Tree view")
	tree := l.tree.Widget()
	tree.SetStripeVisible(model.Lists().IsStripeVisible())
	if model.Lists().IsHeaderVisible() {
		tree.SetHeaderHeight(u)
	} else {
		tree.SetHeaderHeight(0)
	}
	if model.Lists().IsFooterVisible() {
		tree.SetFooterHeight(u)
	} else {
		tree.SetFooterHeight(0)
	}
	guigui.RegisterEventHandler2(l, tree)

	l.treeItems = slices.Delete(l.treeItems, 0, len(l.treeItems))
	l.treeItems = model.Lists().AppendTreeItems(l.treeItems)
	tree.SetItems(l.treeItems)
	context.SetEnabled(&l.tree, model.Lists().Enabled())
	l.tree.SetFixedHeight(6 * u)

	l.listForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &l.listText,
			SecondaryWidget: &l.list,
		},
		{
			PrimaryWidget:   &l.treeText,
			SecondaryWidget: &l.tree,
		},
	})

	// Jump to index
	l.indexText.SetValue("Index")
	l.indexNumberInput.SetMinimumValue(1)
	l.indexNumberInput.SetMaximumValue(model.Lists().ListItemCount())
	l.jumpButton.SetText("Ensure the item is visible")
	l.jumpButton.SetOnDown(func(context *guigui.Context) {
		index := l.indexNumberInput.Value() - 1
		l.list.Widget().EnsureItemVisibleByIndex(index)
		l.list.Widget().SelectItemByIndex(index)
	})

	l.jumpForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &l.indexText,
			SecondaryWidget: &l.indexNumberInput,
		},
		{
			SecondaryWidget: &l.jumpButton,
		},
	})

	// Configurations
	l.showStripeText.SetValue("Show stripe")
	l.showStripeToggle.SetOnValueChanged(func(context *guigui.Context, value bool) {
		model.Lists().SetStripeVisible(value)
	})
	l.showStripeToggle.SetValue(model.Lists().IsStripeVisible())
	l.showHeaderText.SetValue("Show header")
	l.showHeaderToggle.SetOnValueChanged(func(context *guigui.Context, value bool) {
		model.Lists().SetHeaderVisible(value)
	})
	l.showHeaderToggle.SetValue(model.Lists().IsHeaderVisible())
	l.showFooterText.SetValue("Show footer")
	l.showFooterToggle.SetOnValueChanged(func(context *guigui.Context, value bool) {
		model.Lists().SetFooterVisible(value)
	})
	l.showFooterToggle.SetValue(model.Lists().IsFooterVisible())
	l.movableText.SetValue("Enable to move items")
	l.movableToggle.SetValue(model.Lists().Movable())
	l.movableToggle.SetOnValueChanged(func(context *guigui.Context, value bool) {
		model.Lists().SetMovable(value)
	})
	l.enabledText.SetValue("Enabled")
	l.enabledToggle.SetOnValueChanged(func(context *guigui.Context, value bool) {
		model.Lists().SetEnabled(value)
	})
	l.enabledToggle.SetValue(model.Lists().Enabled())

	l.configForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &l.showStripeText,
			SecondaryWidget: &l.showStripeToggle,
		},
		{
			PrimaryWidget:   &l.showHeaderText,
			SecondaryWidget: &l.showHeaderToggle,
		},
		{
			PrimaryWidget:   &l.showFooterText,
			SecondaryWidget: &l.showFooterToggle,
		},
		{
			PrimaryWidget:   &l.movableText,
			SecondaryWidget: &l.movableToggle,
		},
		{
			PrimaryWidget:   &l.enabledText,
			SecondaryWidget: &l.enabledToggle,
		},
	})
	l.listFormPanel.SetContent(&l.listForm)
	l.listFormPanel.SetAutoBorder(true)
	l.listFormPanel.SetContentConstraints(basicwidget.PanelContentConstraintsFixedWidth)

	return nil
}

func (l *Lists) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	u := basicwidget.UnitSize(context)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &l.listFormPanel,
				Size:   guigui.FlexibleSize(1),
			},
			{
				Widget: &l.jumpForm,
			},
			{
				Widget: &l.configForm,
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

func (l *Lists) HandleEvent(context *guigui.Context, targetWidget guigui.Widget, eventArgs any) {
	if targetWidget == &l.list {
		model := context.Model(l, modelKeyModel).(*Model)
		switch eventArgs := eventArgs.(type) {
		case *basicwidget.ListEventArgsItemsMoved:
			idx := model.Lists().MoveListItems(eventArgs.From, eventArgs.Count, eventArgs.To)
			l.list.Widget().SelectItemByIndex(idx)
		case *basicwidget.ListEventArgsItemExpanderToggled:
			model.Lists().SetTreeItemExpanded(eventArgs.Index, eventArgs.Expanded)
		}
	}
}
