package dao

import "fyne.io/fyne/v2/data/binding"

var isTest = true

type SearchReq struct {
	Filter binding.String
	BaseDN binding.String
}

func NewSearchReq() *SearchReq {
	res := &SearchReq{
		Filter: binding.NewString(),
		BaseDN: binding.NewString(),
	}
	if isTest {
		res.Filter.Set("(uid=wanna*)")
		res.BaseDN.Set("dc=wanna,dc=com")
	}
	return res
}
