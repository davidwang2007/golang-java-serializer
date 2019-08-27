package main

import "fmt"
import "io"

//handle JavaFieldIO

//ReadJavaField read java field
func ReadJavaField(jf *JavaField, reader io.Reader, refs []*JavaReferenceObject) error {
	var err error
	if IsPrimType(jf.FieldType) {
		if jf.FieldValue, err = ReadTcPrimFieldValue(jf.FieldType, reader); err != nil {
			return err
		}
	} else if jf.FieldType == TC_OBJ_ARRAY {
		if jf.FieldValue, err = ReadTcArrayFieldValue(jf.FieldType, jf.FieldObjectClassName, reader, refs); err != nil {
			return err
		}
	} else if jf.FieldType == TC_OBJ_OBJECT {
		if jf.FieldValue, err = ReadTcObjFieldValue(jf.FieldType, jf.FieldObjectClassName, reader, refs); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Unexpected field type 0x%x for field name %s", jf.FieldType, jf.FieldName)
	}

	return err
}

//ReadTcPrimFieldValue prim_typecode value
//8大基本类型
func ReadTcPrimFieldValue(fType byte, reader io.Reader) (interface{}, error) {
	switch fType {
	case TC_PRIM_BOOLEAN:
		if b, err := ReadNextByte(reader); err != nil {
			return nil, err
		} else {
			return b == 0x01, nil
		}
	case TC_PRIM_BYTE:
		return ReadNextByte(reader)
	case TC_PRIM_CHAR:
		if c, err := ReadUint16(reader); err != nil {
			return nil, err
		} else {
			//StdLogger.Warn("Got TC_PRIM_CHAR VALUE %s\n", string(c))
			return string(c), nil
		}
	case TC_PRIM_SHORT:
		if s, err := ReadUint16(reader); err != nil {
			return nil, err
		} else {
			return s, nil
		}
	case TC_PRIM_INTEGER:
		if i, err := ReadUint32(reader); err != nil {
			return nil, err
		} else {
			return i, nil
		}
	case TC_PRIM_LONG:
		if l, err := ReadUint64(reader); err != nil {
			return nil, err
		} else {
			return l, nil
		}
	case TC_PRIM_FLOAT:
		if i, err := ReadUint32(reader); err != nil {
			return nil, err
		} else {
			return float32(i), nil
		}
	case TC_PRIM_DOUBLE:
		if l, err := ReadUint64(reader); err != nil {
			return nil, err
		} else {
			return float64(l), nil
		}
	default:
		return nil, fmt.Errorf("Unexpected prim_typecode 0x%x", fType)
	}
}

//ReadTcObjFieldValue read tc object field value
func ReadTcObjFieldValue(fType byte, fieldObjectClassName string, reader io.Reader, refs []*JavaReferenceObject) (interface{}, error) {
	if fType != TC_OBJ_OBJECT {
		return nil, fmt.Errorf("Expected TC_OBJ_OBJECT, but got 0x%x", fType)
	}
	switch fieldObjectClassName {
	case "Ljava/lang/String;":
		return ReadNextTcString(reader, refs)
	default:
		//return nil, fmt.Errorf("Not support field value type classname [%s]", fieldObjectClassName)
		//假设为TC_OBJECT
		jo := &JavaTcObject{}
		if err := jo.Deserialize(reader, refs); err != nil {
			return nil, err
		} else {
			return jo, nil
		}
	}

}

//ReadTcArrayFieldValue read tc object field value
func ReadTcArrayFieldValue(fType byte, fieldObjectClassName string, reader io.Reader, refs []*JavaReferenceObject) (interface{}, error) {
	if fType != TC_OBJ_ARRAY {
		return nil, fmt.Errorf("Expected TC_OBJ_ARRAY , but got 0x%x", fType)
	}
	tcArr := &JavaTcArray{}
	if err := tcArr.Deserialize(reader, refs); err != nil {
		return nil, err
	} else {
		return tcArr, nil
	}

}
