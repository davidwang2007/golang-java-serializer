package main

import "testing"
import "os"

func TestJavaTcArray(t *testing.T) {
	var f *os.File
	var err error

	if f, err = os.Open("d:\\tmp\\serialize-child.data"); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()
	arr := make([]byte, 1<<7) //128

	if _, err = f.Read(arr[:4]); err != nil {
		//to be continued...
		t.Fatalf("Got error %v\n", err)
	}
	refs := make([]*JavaReferenceObject, 1<<7)
	jarr := &JavaTcArray{}
	if err = jarr.Deserialize(f, refs); err != nil {
		t.Fatalf("When deserialize JavaTcArray got %v\n", err)
	}
	t.Logf("Got Tc_ARRAY %v\n", jarr)

}

//TestArraySerialize test JavaTcArray serialize
func TestArraySerialize(t *testing.T) {

	jArr := NewJavaTcArray(uint64(SID_BYTE_ARRAY))

	content := []byte{0x01, 0x02, 0x03, 0x04}
	for _, c := range content {
		jArr.Values = append(jArr.Values, c)
	}

	clz := NewJavaTcClassDesc("[B", uint64(SID_BYTE_ARRAY), SC_SERIALIZABLE)
	jArr.ClassDesc = clz

	var f *os.File
	var err error

	if f, err = os.OpenFile("d:\\tmp\\serialize-go-arr.data", os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()

	if err = SerializeJavaEntity(f, jArr); err != nil {
		t.Fatalf("SerializeJavaEntity got %v\n", err)
	} else {
		t.Logf("SerializeJavaEntity succeed!\n")
	}

}
func TestStringArraySerialize(t *testing.T) {
	/*
		jArr := NewJavaTcArray(uint64(SID_STRING_ARRAY))

		clz := NewJavaTcClassDesc("[Ljava.lang.String;", uint64(SID_STRING_ARRAY), SC_SERIALIZABLE)
		jArr.ClassDesc = clz
		jArr.Values = append(jArr.Values, "a")
		jArr.Values = append(jArr.Values, "b")
		jArr.Values = append(jArr.Values, "c")
	*/
	jArr := NewStringArray([]string{"a", "b", "c"})

	var f *os.File
	var err error

	if f, err = os.OpenFile("d:\\tmp\\serialize-go-arr-1.data", os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()

	if err = SerializeJavaEntity(f, jArr); err != nil {
		t.Fatalf("SerializeJavaEntity got %v\n", err)
	} else {
		t.Logf("SerializeJavaEntity succeed!\n")
	}

}
