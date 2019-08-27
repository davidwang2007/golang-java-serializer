package main

import "io"
import "fmt"

//负责将合适的 ScFlag为 SC_RW_FLAG的路由至自定义的各个JavaSerializer 实现类
//2018-02-01 15:41:51 davidwang2006@aliyun.com

//DeserializeScRwObject
//反序列化 SC_FLAG为 SC_RW_OBJECT 0x03的
//我们从0x78, 0x70 之后真正开始数据的地方读取
func DeserializeScRwObject(reader io.Reader, refs []*JavaReferenceObject, className string) (JavaSerializer, error) {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[DeserializeScRwObject] >>\n")
	defer StdLogger.Debug("[DeserializeScRwObject] <<\n")
	//
	switch className {
	case "java.util.HashMap", "java.util.LinkedHashMap":
		mp := &JavaHashMap{}
		if err := mp.Deserialize(reader, refs); err != nil {
			return nil, err
		} else {
			return mp, nil
		}
	case "java.util.ArrayList":
		lst := &JavaArrayList{}
		if err := lst.Deserialize(reader, refs); err != nil {
			return nil, err
		} else {
			return lst, nil
		}
	case "java.util.LinkedList":
		lst := &JavaLinkedList{}
		if err := lst.Deserialize(reader, refs); err != nil {
			return nil, err
		} else {
			return lst, nil
		}
	default:
		return nil, fmt.Errorf("[DeserializeScRwObject] unexpected className %s, not be supported", className)
	}
}

//ReadNextEle
//read next map entry or list element
func ReadNextEle(reader io.Reader, refs []*JavaReferenceObject) (JavaSerializer, error) {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[ReadNextEle] >>\n")
	defer StdLogger.Debug("[ReadNextEle] <<\n")
	var tp byte //type
	var err error

	if tp, err = ReadNextByte(reader); err != nil {
		return nil, err
	}
	StdLogger.Debug("[ReadNextEle] type is 0x%x\n", tp)
	var js JavaSerializer
	switch tp {
	case TC_STRING:
		js = new(JavaTcString)
	case TC_ARRAY:
		js = &JavaTcArray{}
	case TC_OBJECT:
		js = &JavaTcObject{}
	case TC_REFERENCE:
		if refIndex, err := ReadUint32(reader); err != nil {
			return nil, err
		} else {
			ref := refs[refIndex-INTBASE_WIRE_HANDLE]
			switch ref.RefType {
			case TC_STRING:
				if str, ok := ref.Val.(string); !ok {
					return nil, fmt.Errorf("[JavaHashMap] ref [%v] value should be string type", ref.Val)
				} else {
					tcStr := new(JavaTcString)
					*tcStr = JavaTcString(str)
					return tcStr, nil
				}
			case TC_ARRAY, TC_OBJECT:
				if tempJs, ok := ref.Val.(JavaSerializer); !ok {
					return nil, fmt.Errorf("[JavaHashMap] ref [%v] value should be JavaSerializer type", ref.Val)
				} else {
					return tempJs, nil
				}
			default:
				return nil, fmt.Errorf("[JavaHashMap] unexpected refType 0x%x", ref.RefType)

			}

		}
	default:
		return nil, fmt.Errorf("Unexpected type 0x%x for map entry", tp)
	}
	if err = js.Deserialize(reader, refs); err != nil {
		return nil, err
	}
	return js, nil

}

//SerializeScRwObject
//序列化 SC_FLAG为 SC_RW_OBJECT 0x03的
//我们从0x78, 0x70 之后真正开始数据的地方写入
func SerializeScRwObject(writer io.Writer, refs []*JavaReferenceObject, classDesc *JavaTcClassDesc) error {
	className := classDesc.ClassName
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[SerializeScRwObject] >>\n")
	defer StdLogger.Debug("[SerializeScRwObject] <<\n")
	//
	switch className {
	case "java.util.HashMap", "java.util.LinkedHashMap":
		mp := &JavaHashMap{
			ClassDesc: classDesc,
		}
		if err := mp.Serialize(writer, refs); err != nil {
			return err
		} else {
			return nil
		}
	case "java.util.ArrayList":
		lst := &JavaArrayList{}
		if err := lst.Serialize(writer, refs); err != nil {
			return err
		} else {
			return nil
		}
	case "java.util.LinkedList":
		lst := &JavaLinkedList{}
		if err := lst.Serialize(writer, refs); err != nil {
			return err
		} else {
			return nil
		}
	default:
		return fmt.Errorf("[SerializeScRwObject] unexpected className %s, not be supported", className)
	}
}
