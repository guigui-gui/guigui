// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/guigui-gui/guigui"
	"github.com/guigui-gui/guigui/basicwidget"
)

type Root struct {
	guigui.DefaultWidget

	background basicwidget.Background

	expander1    basicwidget.Expander
	headerText1  basicwidget.Text
	contentText1 basicwidget.Text

	divider1 basicwidget.Divider

	expander2    basicwidget.Expander
	headerText2  basicwidget.Text
	contentText2 basicwidget.Text

	divider2 basicwidget.Divider

	expander3    basicwidget.Expander
	headerText3  basicwidget.Text
	contentText3 basicwidget.Text

	layoutItems []guigui.LinearLayoutItem
}

func (r *Root) Build(context *guigui.Context, adder *guigui.ChildAdder) error {
	adder.AddWidget(&r.background)
	adder.AddWidget(&r.expander1)
	adder.AddWidget(&r.divider1)
	adder.AddWidget(&r.expander2)
	adder.AddWidget(&r.divider2)
	adder.AddWidget(&r.expander3)

	r.headerText1.SetValue("Expander 1")
	r.headerText1.SetBold(true)
	r.headerText1.SetScale(1)
	r.contentText1.SetValue("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
	r.contentText1.SetAutoWrap(true)
	r.contentText1.SetMultiline(true)
	r.contentText1.SetScale(1)
	r.contentText1.SetSelectable(true)
	r.expander1.SetHeaderWidget(&r.headerText1)
	r.expander1.SetContentWidget(&r.contentText1)

	r.headerText2.SetValue("Expander 2")
	r.headerText2.SetBold(true)
	r.headerText2.SetScale(1)
	r.contentText2.SetValue("Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.")
	r.contentText2.SetAutoWrap(true)
	r.contentText2.SetMultiline(true)
	r.contentText2.SetScale(1)
	r.contentText2.SetSelectable(true)
	r.expander2.SetHeaderWidget(&r.headerText2)
	r.expander2.SetContentWidget(&r.contentText2)

	r.headerText3.SetValue("Expander 3")
	r.headerText3.SetBold(true)
	r.headerText3.SetScale(1)
	r.contentText3.SetValue("Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt.")
	r.contentText3.SetAutoWrap(true)
	r.contentText3.SetMultiline(true)
	r.contentText3.SetScale(1)
	r.contentText3.SetSelectable(true)
	r.expander3.SetHeaderWidget(&r.headerText3)
	r.expander3.SetContentWidget(&r.contentText3)

	return nil
}

func (r *Root) Layout(context *guigui.Context, widgetBounds *guigui.WidgetBounds, layouter *guigui.ChildLayouter) {
	layouter.LayoutWidget(&r.background, widgetBounds.Bounds())

	u := basicwidget.UnitSize(context)
	r.layoutItems = slices.Delete(r.layoutItems, 0, len(r.layoutItems))
	r.layoutItems = append(r.layoutItems,
		guigui.LinearLayoutItem{
			Widget: &r.expander1,
		},
		guigui.LinearLayoutItem{
			Widget: &r.divider1,
		},
		guigui.LinearLayoutItem{
			Widget: &r.expander2,
		},
		guigui.LinearLayoutItem{
			Widget: &r.divider2,
		},
		guigui.LinearLayoutItem{
			Widget: &r.expander3,
		},
	)
	(guigui.LinearLayout{
		Direction: guigui.LayoutDirectionVertical,
		Items:     r.layoutItems,
		Padding: guigui.Padding{
			Start:  u,
			Top:    u,
			End:    u,
			Bottom: u,
		},
	}).LayoutWidgets(context, widgetBounds.Bounds(), layouter)
}

func main() {
	op := &guigui.RunOptions{
		Title: "Expander",
	}
	if err := guigui.Run(&Root{}, op); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
