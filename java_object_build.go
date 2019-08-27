package main

//NewJavaTcArray new java tc array
func NewJavaTcArray(serialVersionUID uint64) *JavaTcArray {
	jarr := &JavaTcArray{
		SerialVersionUID: serialVersionUID,
	}
	jarr.Values = make([]interface{}, 0, 1<<7)
	return jarr
}

//NewJavaTcObject new java tc object
func NewJavaTcObject(serialVersionUID uint64) *JavaTcObject {
	jo := &JavaTcObject{}
	jo.Classes = make([]*JavaTcClassDesc, 0, 4)
	jo.SerialVersionUID = serialVersionUID
	return jo
}

//NewJavaTcClassDesc
func NewJavaTcClassDesc(className string, serialVersionUID uint64, scFlag byte) *JavaTcClassDesc {
	jcd := &JavaTcClassDesc{}
	jcd.SerialVersionUID = serialVersionUID
	jcd.Fields = make([]*JavaField, 0, 8)
	jcd.ClassName = className
	jcd.ScFlag = scFlag
	return jcd
}

//NewJavaField
func NewJavaField(tp byte, name string, v interface{}) *JavaField {
	jf := &JavaField{
		FieldType:  tp,
		FieldName:  name,
		FieldValue: v,
	}
	return jf
}

//NewStringJavaField new String java field
func NewStringJavaField(name string, v string) *JavaField {
	jfb := NewJavaField(TC_OBJ_OBJECT, name, v)
	jfb.FieldObjectClassName = "java.lang.String"
	return jfb
}

//NewObjectJavaField new object java field
func NewObjectJavaField(objClassName string, name string, v interface{}) *JavaField {
	jfb := NewJavaField(TC_OBJ_OBJECT, name, v)
	jfb.FieldObjectClassName = objClassName
	return jfb
}

func NewJavaTcString(str string) *JavaTcString {
	jts := new(JavaTcString)
	*jts = (JavaTcString)(str)
	return jts
}

//NewByteArray
func NewByteArray(items []byte) *JavaTcArray {
	jArr := NewJavaTcArray(uint64(SID_BYTE_ARRAY))
	for _, c := range items {
		jArr.Values = append(jArr.Values, c)
	}

	clz := NewJavaTcClassDesc("[B", uint64(SID_BYTE_ARRAY), SC_SERIALIZABLE)
	jArr.ClassDesc = clz

	return jArr

}

//NewStringArray
func NewStringArray(items []string) *JavaTcArray {
	jArr := NewJavaTcArray(uint64(SID_STRING_ARRAY))
	for _, c := range items {
		jArr.Values = append(jArr.Values, c)
	}

	clz := NewJavaTcClassDesc("[Ljava.lang.String;", uint64(SID_STRING_ARRAY), SC_SERIALIZABLE)
	jArr.ClassDesc = clz

	return jArr

}
