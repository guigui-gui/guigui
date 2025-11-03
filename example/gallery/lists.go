// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025 The Guigui Authors

package main

import (
	"image"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Lists struct {
	guigui.DefaultWidget

	listFormPanel    basicwidget.Panel
	listForm         basicwidget.Form
	listText         basicwidget.Text
	list             guigui.WidgetWithSize[*basicwidget.List[int]]
	treeText         basicwidget.Text
	tree             guigui.WidgetWithSize[*basicwidget.List[int]]
	dropdownListText basicwidget.Text
	dropdownList     guigui.WidgetWithSize[*basicwidget.DropdownList[int]]

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

	listItems         []basicwidget.ListItem[int]
	treeItems         []basicwidget.ListItem[int]
	dropdownListItems []basicwidget.DropdownListItem[int]
}

func (l *Lists) AddChildren(context *guigui.Context, adder *guigui.ChildAdder) {
	adder.AddChild(&l.listFormPanel)
	adder.AddChild(&l.configForm)
}

func (l *Lists) Update(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
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
	list.SetOnItemsMoved(func(from, count, to int) {
		idx := model.Lists().MoveListItems(from, count, to)
		list.SelectItemByIndex(idx)
	})

	l.listItems = slices.Delete(l.listItems, 0, len(l.listItems))
	l.listItems = model.lists.AppendListItems(l.listItems)
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
	tree.SetOnItemExpanderToggled(func(index int, expanded bool) {
		model.Lists().SetTreeItemExpanded(index, expanded)
	})

	l.treeItems = slices.Delete(l.treeItems, 0, len(l.treeItems))
	l.treeItems = model.lists.AppendTreeItems(l.treeItems)
	tree.SetItems(l.treeItems)
	context.SetEnabled(&l.tree, model.Lists().Enabled())
	l.tree.SetFixedHeight(6 * u)

	// Dropdown list
	l.dropdownListText.SetValue("Dropdown list")
	dropdownList := l.dropdownList.Widget()
	l.dropdownListItems = slices.Delete(l.dropdownListItems, 0, len(l.dropdownListItems))
	l.dropdownListItems = model.lists.AppendDropdownListItems(l.dropdownListItems)
	dropdownList.SetItems(l.dropdownListItems)
	context.SetEnabled(&l.dropdownList, model.Lists().Enabled())

	l.listForm.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &l.listText,
			SecondaryWidget: &l.list,
		},
		{
			PrimaryWidget:   &l.treeText,
			SecondaryWidget: &l.tree,
		},
		{
			PrimaryWidget:   &l.dropdownListText,
			SecondaryWidget: &l.dropdownList,
		},
	})

	// Configurations
	l.showStripeText.SetValue("Show stripe")
	l.showStripeToggle.SetOnValueChanged(func(value bool) {
		model.Lists().SetStripeVisible(value)
	})
	l.showStripeToggle.SetValue(model.Lists().IsStripeVisible())
	l.showHeaderText.SetValue("Show header")
	l.showHeaderToggle.SetOnValueChanged(func(value bool) {
		model.Lists().SetHeaderVisible(value)
	})
	l.showHeaderToggle.SetValue(model.Lists().IsHeaderVisible())
	l.showFooterText.SetValue("Show footer")
	l.showFooterToggle.SetOnValueChanged(func(value bool) {
		model.Lists().SetFooterVisible(value)
	})
	l.showFooterToggle.SetValue(model.Lists().IsFooterVisible())
	l.movableText.SetValue("Enable to move items")
	l.movableToggle.SetValue(model.Lists().Movable())
	l.movableToggle.SetOnValueChanged(func(value bool) {
		model.Lists().SetMovable(value)
	})
	l.enabledText.SetValue("Enabled")
	l.enabledToggle.SetOnValueChanged(func(value bool) {
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

func (l *Lists) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, widget guigui.Widget) image.Rectangle {
	u := basicwidget.UnitSize(context)
	return (guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items: []guigui.LinearLayoutItem{
			{
				Widget: &l.listFormPanel,
				Size:   guigui.FlexibleSize(1),
			},
			{
				Widget: &l.configForm,
			},
		},
		Gap: u / 2,
	}).WidgetBounds(context, widgetBounds.Bounds().Inset(u/2), widget)
}
