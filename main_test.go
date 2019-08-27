package main

import "testing"
import "os"
import "reflect"
import "math"
import "encoding/json"

func TestSlice(t *testing.T) {

	arr := make([]byte, 0, 10)
	t.Logf("slice is %v\n", arr[:10])
	changeSlice(arr[:0])
	t.Logf("slice is %v\n", arr[:10])

	str := "Ljava.lang.String;"
	switch str {
	case "Ljava.lang.String;":
		t.Logf("str is Ljava.lang.String;\n")
	default:
		t.Logf("str is not Ljava.lang.String;\n")
	}

}

func changeSlice(arr []byte) {
	arr = append(arr, 0x01)
}

func TestJavaTcObject(t *testing.T) {
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
	jo := &JavaTcObject{}
	if err = jo.Deserialize(f, refs); err != nil {
		t.Fatalf("When deserialize JavaTcObject got %v\n", err)
	}
	t.Logf("Got Tc_OBJECT %v\n", jo)
	rv := reflect.ValueOf(jo)
	if rv.Kind() == reflect.Ptr {
		t.Logf("jo type is %s\n", rv.Elem().Type().Name())
	} else {
		t.Logf("jo type is %s\n", rv.Type().Name())
	}

}
func TestJavaDeserialize(t *testing.T) {

	var f *os.File
	var err error

	if f, err = os.Open("d:\\tmp\\serialize-child.data"); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()

	var v JavaSerializer

	if v, err = DeserializeStream(f); err != nil {
		t.Fatalf("When Deserialize stream, got error %v\n", err)
	} else {
		if bs, err := json.MarshalIndent(v.JsonMap(), "", "  "); err != nil {
			t.Fatalf("Json error %v\n", err)
		} else {
			t.Logf("Deserialize stream got\n %s\n", string(bs))
		}
		//t.Logf("Deserialize stream got %v\n", v.JsonMap())
	}

}

//TestLong it's ok
//we must declare it to hold the number before using it
//davidwang2006@aliyun.com 2018-02-01 11:17:24
func TestLong(t *testing.T) {
	var it int64 = -3665804199014368530
	var uit uint64 = uint64(it)
	var it2 int64 = int64(uit)
	t.Logf("%d %x %d", it, uit, it2)
	var i32 int32 = 0x3f400000
	var f32 float32 = math.Float32frombits(uint32(i32)) //float32(i32)
	t.Logf("%d %f", i32, f32)
	buff := make([]byte, 0, 4)
	t.Logf("buff is %v\n", buff)
	t.Logf("buff is %v\n", buff[:4]) //4 is okay
}

//TestObjectSerialize test object serialize object to stream
func TestObjectSerialize(t *testing.T) {

	jo := NewJavaTcObject(1)
	clz := NewJavaTcClassDesc("com.david.test.serialize.D", 1, 0x02)
	jfa := NewJavaField(TC_PRIM_INTEGER, "a", 1)
	jfb := NewJavaField(TC_OBJ_OBJECT, "b", "abcdefg")
	jfb.FieldObjectClassName = "java.lang.String"
	clz.AddField(jfa)
	clz.AddField(jfb)
	clz.SortFields()

	jo.AddClassDesc(clz)

	var f *os.File
	var err error

	if f, err = os.OpenFile("d:\\tmp\\serialize-go.data", os.O_CREATE|os.O_TRUNC, 0755); err != nil {
		t.Fatalf("got error when open file %v\n", err)
	}
	defer f.Close()

	if err = SerializeJavaEntity(f, jo); err != nil {
		t.Fatalf("SerializeJavaEntity got %v\n", err)
	} else {
		t.Logf("SerializeJavaEntity succeed!\n")
	}
}
