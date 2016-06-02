package extractor

import (
	"testing"
)

func TestFindAttribute(t *testing.T) {
	expression := "#J_show_list ul li @index= 3"
	val, ok := FindIntAttribute("index", expression)
	if ok {
		t.Log(val)
	}
}

func TestFindAGroups(t *testing.T) {
	url := "http://www.baidu.com/[$allLink]/aaa/[$all2]"
	vs := FindGroupsByIndex("\\[\\$([^\\]]+)\\]", url, 1)
	t.Log(vs)
}

func TestCleanAttribute(t *testing.T) {
	expression := "#J_show_list ul li @index= 3 "
	val := CleanAttribute("index", expression)
	t.Log(val)
}

func TestFilterJSONP(t *testing.T) {
	content :=
		`
		jquery11({"code":"11"})
	`
	val := FilterJSONP(string(content))
	t.Log(val)
}
