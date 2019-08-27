package main

import "testing"
import "sort"
import "unicode/utf8"

func TestFieldSort(t *testing.T) {
	jfs := make([]*JavaField, 4)
	jfs[0] = &JavaField{
		FieldName: "a",
	}
	jfs[1] = &JavaField{
		FieldName: "z",
	}
	jfs[2] = &JavaField{
		FieldName: "bb",
	}
	jfs[3] = &JavaField{
		FieldName: "aa",
	}
	t.Logf("origin is %v\n", jfs)
	sort.Sort(JFByFieldName(jfs))
	t.Logf("after is %v\n", jfs)

}

func TestRune(t *testing.T) {
	r := rune('你')
	bs := make([]byte, 4)
	utf8.EncodeRune(bs, r)
	t.Logf("%s rune is %+q\n", r, r)
	t.Logf("bs is %v\n", bs)
}
