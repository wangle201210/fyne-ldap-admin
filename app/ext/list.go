package ext

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type List struct {
	widget.List
}

func (x *List) MinSize() fyne.Size {
	org := x.List.MinSize()
	if org.Width < 200 {
		org.Width = 200
	}
	if org.Height < 400 {
		org.Height = 400
	}
	return org
}

func NewList(length func() int, createItem func() fyne.CanvasObject, updateItem func(widget.ListItemID, fyne.CanvasObject)) *List {
	list := &List{List: *widget.NewList(length, createItem, updateItem)}
	return list
}
