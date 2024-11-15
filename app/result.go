package app

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (x *LdapAdmin) ResultShow() {
	showOne := widget.NewLabel("--")
	var items []fyne.CanvasObject
	for _, data := range x.data {
		items = append(items, widget.NewButton(data.DN, func() {
			x.search.selectData = data
			showOne.SetText(x.getShowData())
			showOne.Refresh()
			x.result.Refresh()
		}))
	}
	if len(x.data) == 0 {
		return
	}
	x.search.selectData = x.data[0]
	// 搜索按钮
	nextButton := widget.NewButton("下一页", x.SearchNextPage)
	x.result.Add(nextButton)
	size := fyne.Size{Width: 200, Height: 400}
	left := container.NewVScroll(container.NewVBox(items...))
	left.Resize(size)
	showOne.SetText(x.getShowData())
	showOne.Refresh()
	right := container.NewVBox(showOne)
	right.Resize(size)
	split := container.NewHSplit(left, right)
	x.result.Add(split)
	x.result.Refresh()
}

func (x *LdapAdmin) getShowData() (result string) {
	entry := x.search.selectData
	result += fmt.Sprintf("dn: %s\n", entry.DN)
	for _, e := range entry.Attributes {
		result += fmt.Sprintf("%s: %s\n", e.Name, e.Values)
	}
	result += "\n"
	return result
}
