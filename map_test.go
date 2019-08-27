package main

import "testing"
import "os"

func TestHashMap(t *testing.T) {
	items := make([]interface{}, 4)
	items[0] = "a"
	items[1] = "b"
	items[2] = "c"
	items[3] = "d"

	clzDesc := GenerateHashMapClassDesc(items)
	jo := NewJavaTcObject(SID_HASH_MAP)
	jo.AddClassDesc(clzDesc)

	var f *os.File
	var err error

	if f, err = os.OpenFile("d:\\tmp\\serialize-go-map.data", os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()

	if err = SerializeJavaEntity(f, jo); err != nil {
		t.Fatalf("SerializeJavaEntity got %v\n", err)
	} else {
		t.Logf("SerializeJavaEntity succeed!\n")
	}
}

func TestLinkedHashMap(t *testing.T) {
	items := make([]interface{}, 4)
	items[0] = "a"
	items[1] = "b"
	items[2] = "c"
	items[3] = "d"

	clzDesc := GenerateHashMapClassDesc(items)
	jo := NewJavaTcObject(SID_LINKED_HASH_MAP)
	jo.AddClassDesc(GenerateLinkedHashMapClassDesc())
	jo.AddClassDesc(clzDesc)

	var f *os.File
	var err error

	if f, err = os.OpenFile("d:\\tmp\\serialize-go-linkedmap.data", os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()

	if err = SerializeJavaEntity(f, jo); err != nil {
		t.Fatalf("SerializeJavaEntity got %v\n", err)
	} else {
		t.Logf("SerializeJavaEntity succeed!\n")
	}
}
func TestLinkedHashMap2(t *testing.T) {
	jo := NewLinkedHashMap(map[string]interface{}{"a": "b", "c": "d"})
	var f *os.File
	var err error

	if f, err = os.OpenFile("d:\\tmp\\serialize-go-linkedmap2.data", os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()

	if err = SerializeJavaEntity(f, jo); err != nil {
		t.Fatalf("SerializeJavaEntity got %v\n", err)
	} else {
		t.Logf("SerializeJavaEntity succeed!\n")
	}
}
