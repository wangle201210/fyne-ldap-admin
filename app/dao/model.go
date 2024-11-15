package dao

import "fyne.io/fyne/v2/data/binding"

type SearchReq struct {
	Filter binding.String
	BaseDN binding.String
}

func NewSearchReq() *SearchReq {
	res := &SearchReq{
		Filter: binding.NewString(),
		BaseDN: binding.NewString(),
	}
	return res
}
