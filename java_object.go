package main

import "fmt"
import "math"
import "bytes"
import "reflect"
import "io"
import "sort"
import "strings"
import "encoding/binary"
import "encoding/json"

// https://courses.cs.washington.edu/courses/cse341/98au/java/jdk1.2beta4/docs/guide/serialization/spec/protocol.doc5.html
//
//
// newClass:
// 	TC_CLASS classDesc newHandle
// newClassDesc:
// 	TC_CLASSDESC className serialVersionUID newHandle classDescInfo
// newArray:
// 	TC_ARRAY classDesc newHandle (int)<size> values[size]
// newObject:
// 	TC_OBJECT classDesc newHandle classdata[]      // data for each class
// newString:
// 	TC_STRING newHandle (utf)
//
// 翻译成如下：
// newObject:
// 	TC_OBJECT (TC_CLASSDESC className serialVersionUID newHandle classDescInfo) newHandle classdata[]
//
// newArray:
// 	TC_ARRAY (TC_CLASSDESC className serialVersionUID newHandle classDescInfo) newHandle (int)<size> values[size]
//
// 即TC_OBJECT, TC_ARRAY分别最低分产生二个handle, 即其 TC_CLASSDESC 1个, TC_CLASSDESC 结束之后会产生1个
//
// newHandle会在TC_CLASSDESC, TC_OBJECT, TC_ARRAY, TC_STRING, TC_CLASS 后分别会有一个，但常用的通常是TC_STRING
// 以协议的角度来看0x78,0x70后必有一个newHandle

//Java Field
type JavaField struct {
	FieldType            byte        //field type; prim_typecode, obj_typecode,其值为 TC_PRIM_xxx TC_OBJECT_xx 之一
	FieldName            string      //field name
	FieldOwnerScFlag     byte        //存储所属class的SC_ 序列化指示符,以区分当SC_RW_OBJECT时调用个性化序列化实体类进行操作
	FieldObjectClassName string      //当field type为 obj_typecode时会有此字段
	FieldValue           interface{} //field value
}

func (jf *JavaField) String() string {
	return fmt.Sprintf("type: 0x%x, name: %s, flag: 0x%x, class: %s", jf.FieldType, jf.FieldName, jf.FieldOwnerScFlag, jf.FieldObjectClassName)
}

type JFByFieldName []*JavaField

func (jf JFByFieldName) Len() int {
	return len(jf)
}
func (jf JFByFieldName) Swap(i, j int) {
	jf[i], jf[j] = jf[j], jf[i]
}
func (jf JFByFieldName) Less(i, j int) bool {
	return jf[i].FieldName < jf[j].FieldName
}

//JavaTcClassDesc represent java tc class desc
type JavaTcClassDesc struct {
	ClassName        string //classname
	ScFlag           byte   //Sc flag, indicate serializable mechanism, current support TC_RW_OBJECT, SC_SERIALIZABLE
	SerialVersionUID uint64 // serialVersionUID
	//newHandle
	Fields  []*JavaField  //it's fields
	RwDatas []interface{} //for SC_RW_OBJECT CUSTOM WRITER
}

//SortFields sort fields by name to generate class desc fields description
func (classDesc *JavaTcClassDesc) SortFields() {
	if classDesc.Fields == nil {
		return
	}
	sort.Sort(JFByFieldName(classDesc.Fields))
}

//AddField
func (classDesc *JavaTcClassDesc) AddField(jf *JavaField) {
	classDesc.Fields = append(classDesc.Fields, jf)
}

//JavaTcClass represent java tc_class
//it is rarely used
type JavaTcClass struct {
	ClassDesc JavaTcClassDesc //class desc
	//newHandle
}

//JavaTcArray  represent java tc array object
type JavaTcArray struct {
	ClassDesc *JavaTcClassDesc //class desc
	//newHandle
	SerialVersionUID uint64        // serialVersionUID
	Values           []interface{} //values [size]
	JsonData         []interface{}
}

//JavaTcObject  represent java tc object
type JavaTcObject struct {
	Classes          []*JavaTcClassDesc //it's classes, including the parent class
	SerialVersionUID uint64             // serialVersionUID
	JsonData         interface{}        //即协议中的classdata[]; map, slice, 8大基本类型或包装类型以及String
}

type JavaTcString string

// Deserialize implements JavaSerializer
func (tcStr *JavaTcString) Deserialize(reader io.Reader, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcString] Deserialize >> \n")
	defer StdLogger.Debug("[JavaTcString] Deserialize << \n")
	buff := make([]byte, 4)
	if _, err := reader.Read(buff[:1]); err != nil {
		return err
	}
	var strLen uint16
	var err error
	switch buff[0] {
	case TC_REFERENCE:
		if refIndex, err := ReadUint32(reader); err != nil {
			return err
		} else {
			ref := refs[refIndex-INTBASE_WIRE_HANDLE]
			if ref.RefType != TC_STRING {
				return fmt.Errorf("[JavaTcString] Expect [%d] RefType TC_STRING, but 0x%x", refIndex-INTBASE_WIRE_HANDLE, ref.RefType)
			} else {
				if str, ok := ref.Val.(string); !ok {
					return fmt.Errorf("[JavaTcString] ref [%d] should be string, but %v", refIndex-INTBASE_WIRE_HANDLE, ref.Val)
				} else {
					*tcStr = JavaTcString(str)
				}
				return nil
			}
		}
	case TC_STRING:
		if strLen, err = ReadUint16(reader); err != nil {
			return err
		}
		fallthrough
	default: //假设头一个字节已消耗
		if _, err := reader.Read(buff[1:2]); err != nil {
			return err
		}
		strLen = binary.BigEndian.Uint16(buff[:2])

		if str, err := ReadUTFString(reader, int(strLen)); err != nil {
			return err
		} else {
			*tcStr = JavaTcString(str)
			AddReference(refs, TC_STRING, str)
			return nil
		}

	}
}

//Serialize JavaTcString to stream
func (tcStr *JavaTcString) Serialize(write io.Writer, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcString] Serialize >> \n")
	defer StdLogger.Debug("[JavaTcString] Serialize << \n")
	var refIndex int = -1
	for i := 0; i < len(refs); i++ {
		ref := refs[i]
		if ref == nil {
			break
		}
		if ref.RefType != TC_STRING {
			continue
		}
		if str, ok := ref.Val.(string); ok && string(*tcStr) == str {
			refIndex = i
			break
		} else if tsp, ok := ref.Val.(*JavaTcString); ok && *tsp == *tcStr {
			refIndex = i
			break
		}
	}
	var err error
	buff := make([]byte, 5)
	if refIndex >= 0 {
		buff[0] = TC_REFERENCE
		refIndex += INTBASE_WIRE_HANDLE
		binary.BigEndian.PutUint32(buff[1:5], uint32(refIndex))
		_, err = write.Write(buff[:5])
		return err
	}

	//write tc_string, len,
	strBs := ([]byte)(*tcStr)
	buff[0] = TC_STRING
	binary.BigEndian.PutUint16(buff[1:3], uint16(len(strBs)))
	if _, err = write.Write(buff[:3]); err != nil {
		return err
	}
	if _, err = write.Write(strBs); err != nil {
		return err
	}
	//add it to refs
	AddReference(refs, TC_STRING, tcStr)

	return nil
}

// JsonMap implements JavaSerializer
func (tcStr *JavaTcString) JsonMap() interface{} {
	return *tcStr
}

//we keep JavaTcString just as string type

//Deserialize stream to JavaTcClassDesc
//一个classDesc从TC_CLASSDESC开始，以TC_ENDBLOCKDATA终
//一个TC_OBJECT包含多个TC_CLASSDESC
func (classDesc *JavaTcClassDesc) Deserialize(reader io.Reader, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcClassDesc] >> ++ BEGIN\n")
	defer StdLogger.Debug("[JavaTcClassDesc] << --END\n")
	//TC_CLASSDESC
	var buff = make([]byte, 4)
	var err error
	var classNameLen uint16
	if _, err = reader.Read(buff[:2]); err != nil {
		return err
	}
	if TC_CLASSDESC == buff[0] { //证明这个开头的指示TC_CLASSDESC未被消费掉
		if _, err = reader.Read(buff[2:3]); err != nil {
			return err
		}
		classNameLen = binary.BigEndian.Uint16(buff[1:3])
	} else if TC_REFERENCE == buff[0] { //表示引用了另一个CLASSDESC
		//读剩下的3个字节，后二个字节表示refIndex
		if _, err = reader.Read(buff[:3]); err != nil {
			return err
		} else {
			refIndex := binary.BigEndian.Uint16(buff[1:3])
			ref := refs[int(refIndex)]
			if ref.RefType != TC_CLASSDESC {
				return fmt.Errorf("[JavaTcClassDesc] Expect ref [%d] type TC_CLASSDESC, but 0x%x", refIndex, ref.RefType)
			} else if cdp, ok := ref.Val.(*JavaTcClassDesc); !ok {
				return fmt.Errorf("[JavaTcClassDesc] Expect ref[%d] val *JavaTcClassDesc, but %v", refIndex, ref.Val)
			} else {
				classDesc.ClassName = cdp.ClassName
				classDesc.SerialVersionUID = cdp.SerialVersionUID
				classDesc.Fields = cdp.Fields
				return nil
			}
		}
	} else {
		//标志已被消费掉
		classNameLen = binary.BigEndian.Uint16(buff[:2])
	}
	StdLogger.Debug("[JavaTcClassDesc] TRY TO Read classDesc.className, len=%d\n", classNameLen)
	if classDesc.ClassName, err = ReadUTFString(reader, int(classNameLen)); err != nil {
		return err
	}
	StdLogger.Debug("[JavaTcClassDesc] classDesc.className is [%s]\n", classDesc.ClassName)
	if classDesc.SerialVersionUID, err = ReadUint64(reader); err != nil {
		return err
	}
	//after read serialVersionUID newHandle should be added
	AddReference(refs, TC_CLASSDESC, classDesc)

	//next byte
	//various flag, This particular flag says that the object supports serialization.
	//current time just support 0x02 SC_SERIALIZABLE & some SC_RW_OBJECT
	if sc, err := ReadNextByte(reader); err != nil {
		return err
	} else if sc != SC_SERIALIZABLE && sc != SC_RW_OBJECT {
		return fmt.Errorf("[JavaTcClassDesc] Cannot handle Serializable flag 0x%x", sc)
	} else {
		classDesc.ScFlag = sc
	}
	//number of fields in this class
	if numberOfFields, err := ReadUint16(reader); err != nil {
		return err
	} else {
		StdLogger.Debug("[JavaTcClassDesc] %s has %d fields\n", classDesc.ClassName, numberOfFields)
		classDesc.Fields = make([]*JavaField, int(numberOfFields))
		for i := 0; i < int(numberOfFields); i++ {
			if classDesc.Fields[i], err = ReadNextJavaField(reader, refs); err != nil {
				return err
			} else {
				classDesc.Fields[i].FieldOwnerScFlag = classDesc.ScFlag
			}
		}
	}
	//NOW IT SHOULD BE TC_ENDBLOCKDATA
	if b, err := ReadNextByte(reader); err != nil {
		return err
	} else if b != TC_ENDBLOCKDATA {
		return fmt.Errorf("[JavaTcClassDesc] Expect TC_ENDBLOCKDATA 0x78, but got 0x%x", b)
	}
	return nil
}

//Serialize serialize JavaTcClassDesc to stream
//2018-02-02 11:15:09
func (classDesc *JavaTcClassDesc) Serialize(writer io.Writer, refs []*JavaReferenceObject) error {
	//judge if exists in ref
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcClassDesc] Serialize >> \n")
	defer StdLogger.Debug("[JavaTcClassDesc] Serialize << \n")
	var refIndex int = -1
	for i := 0; i < len(refs); i++ {
		ref := refs[i]
		if ref == nil {
			break
		}
		if ref.RefType != TC_CLASSDESC {
			continue
		}
		if tcdp, ok := ref.Val.(*JavaTcClassDesc); ok && tcdp.SerialVersionUID == classDesc.SerialVersionUID {
			refIndex = i
			break
		}
	}
	var err error
	buff := make([]byte, 8)
	if refIndex >= 0 {
		buff[0] = TC_REFERENCE
		refIndex += INTBASE_WIRE_HANDLE
		binary.BigEndian.PutUint32(buff[1:5], uint32(refIndex))
		_, err = writer.Write(buff[:5])
		return err
	}
	//ordinary serialize
	//0x72
	buff[0] = TC_CLASSDESC
	//classname length uint16
	classNameArr := ([]byte)(classDesc.ClassName)
	binary.BigEndian.PutUint16(buff[1:3], uint16(len(classNameArr)))
	if _, err = writer.Write(buff[:3]); err != nil { // TC_CLASSDESC & classNameLen sum 3 bytes
		return err
	}
	//classname
	if _, err = writer.Write(classNameArr); err != nil {
		return err
	}
	//serialVersionUID
	binary.BigEndian.PutUint64(buff[:8], classDesc.SerialVersionUID)
	if _, err = writer.Write(buff[:8]); err != nil {
		return err
	}
	//newHandle
	AddReference(refs, TC_CLASSDESC, classDesc)
	//SC_FLAG
	buff[0] = classDesc.ScFlag
	if _, err = writer.Write(buff[:1]); err != nil {
		return err
	}
	//field count
	if classDesc.Fields == nil {
		classDesc.Fields = make([]*JavaField, 0)
	}
	var fieldCount int = len(classDesc.Fields)
	binary.BigEndian.PutUint16(buff[:2], uint16(fieldCount))
	if _, err = writer.Write(buff[:2]); err != nil {
		return err
	}
	//writer all fields type declaration
	//classDesc.SortFields()
	for i, jf := range classDesc.Fields {
		StdLogger.Debug("[JavaTcClassDesc] Serialize field [%d] %v \n", i, jf)
		//1byte type + 2 byte len + n byte fieldName
		buff[0] = jf.FieldType
		fieldNameArr := ([]byte)(jf.FieldName)
		binary.BigEndian.PutUint16(buff[1:3], uint16(len(fieldNameArr)))
		fObjNameArr := ([]byte)(jf.FieldObjectClassName)
		var modifiedName string = jf.FieldObjectClassName
		if _, err = writer.Write(buff[:3]); err != nil {
			return err
		}
		if _, err = writer.Write(fieldNameArr); err != nil {
			return err
		}
		switch jf.FieldType {
		case TC_PRIM_BYTE, TC_PRIM_BOOLEAN, TC_PRIM_CHAR, TC_PRIM_SHORT, TC_PRIM_INTEGER, TC_PRIM_LONG, TC_PRIM_FLOAT, TC_PRIM_DOUBLE:
		//八种基本类型不用再放任何东西
		case TC_OBJ_ARRAY: // '['
			//还要再写 jf.FieldObjectClassName 的 TC_STRING
			//get the FieldObjectClassName 考虑到 java.lang.String ->
			var b0 byte = fObjNameArr[0]
			if b0 != TC_OBJ_ARRAY {
				//要prefix上 [
				if len(fObjNameArr) == 1 {
					modifiedName = string([]byte{TC_OBJ_ARRAY, b0})
				} else {
					//替换.为/ 最后加;
					modifiedName = fmt.Sprintf("[L%s;", strings.Replace(jf.FieldObjectClassName, ".", "/", -1))
				}
				StdLogger.Debug("[JavaTcClassDesc] Serialize modify field name %s » %s \n", jf.FieldObjectClassName, modifiedName)
			}
			//write it out
			tcString := new(JavaTcString)
			*tcString = (JavaTcString)(modifiedName)
			if err = tcString.Serialize(writer, refs); err != nil {
				return err
			}
		case TC_OBJ_OBJECT: // 'L'
			//还要再写 jf.FieldObjectClassName 的 TC_STRING
			//get the FieldObjectClassName 考虑到 java.lang.String ->
			var b0 byte = fObjNameArr[0]
			if b0 != TC_OBJ_OBJECT {
				//要prefix上 L
				//替换.为/ 最后加;
				modifiedName = fmt.Sprintf("L%s;", strings.Replace(jf.FieldObjectClassName, ".", "/", -1))
				StdLogger.Debug("[JavaTcClassDesc] Serialize modify field name %s » %s \n", jf.FieldObjectClassName, modifiedName)
			}
			//write it out
			tcString := new(JavaTcString)
			*tcString = (JavaTcString)(modifiedName)
			if err = tcString.Serialize(writer, refs); err != nil {
				return err
			}
		}
	}
	//写完Field后写TC_ENDBLOCKDATA

	buff[0] = TC_ENDBLOCKDATA
	if _, err = writer.Write(buff[:1]); err != nil {
		return err
	}

	return nil
}

//Deserialize deserialize stream to tc object
func (jo *JavaTcObject) Deserialize(reader io.Reader, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcObject] >> ++BEGIN\n")
	defer StdLogger.Debug("[JavaTcObject] << --END\n")
	//firstly, analysis all tc_classdesc
	//TC_OBJECT
	var buff = make([]byte, 4)
	var err error
	if _, err = reader.Read(buff[:1]); err != nil {
		return err
	}
	if TC_REFERENCE == buff[0] { //表示引用了另一个TC_OBJECT
		//读剩下的3个字节，后二个字节表示refIndex
		if _, err = reader.Read(buff[:4]); err != nil {
			return err
		} else {
			refIndex := binary.BigEndian.Uint16(buff[2:4])
			ref := refs[int(refIndex)]
			if ref.RefType != TC_OBJECT {
				return fmt.Errorf("[JavaTcObject] Expect ref [%d] type TC_OBJECT, but 0x%x", refIndex, ref.RefType)
			} else if jop, ok := ref.Val.(*JavaTcObject); !ok {
				return fmt.Errorf("[JavaTcObject] Expect ref[%d] val *JavaTcObject, but %v", refIndex, ref.Val)
			} else {
				jo.Classes = jop.Classes
				jo.JsonData = jop.JsonData
				return nil
			}
		}
	} else if TC_OBJECT == buff[0] { //证明开头的tc_object未被消费，则再读下一个
		if _, err = reader.Read(buff[:1]); err != nil {
			return err
		}
	}

	//now begin tc_classdesc
	jo.Classes = make([]*JavaTcClassDesc, 0, 1<<7)

out:
	for {
		switch buff[0] {
		case TC_REFERENCE:
			//在TC_OBJECT 之后遇到 TC_REFERENCE后，就不会有0x78,0x70结束符了
			if _, err = reader.Read(buff[:4]); err != nil {
				StdLogger.Debug("[JavaTcObject] try to get classDesc ref failed: %v\n", err)
			} else {
				refIndex := int(binary.BigEndian.Uint32(buff[:4]))
				ref := refs[refIndex-INTBASE_WIRE_HANDLE]
				if ref.RefType != TC_CLASSDESC {
					return fmt.Errorf("Expected TC_CLASSDESC @ ref [%d]", refIndex-INTBASE_WIRE_HANDLE)
				} else if tcd, ok := ref.Val.(*JavaTcClassDesc); ok {
					jo.Classes = append(jo.Classes, tcd)
					break out
				} else {
					return fmt.Errorf("Unexpected error when deserialize TC_OBJECT, read TC_CLASSDESC, ref=%v", ref)
				}

			}
		case TC_CLASSDESC:
			tcs := &JavaTcClassDesc{}
			StdLogger.Debug("[JavaTcObject] try to get classDesc [%d]\n", len(jo.Classes))
			jo.Classes = append(jo.Classes, tcs)
			if err := tcs.Deserialize(reader, refs); err != nil {
				return err
			}
		case TC_NULL:
			//newHandle
			AddReference(refs, TC_OBJECT, jo)
			break out
		default:
			return fmt.Errorf("[JavaTcObject] Expected TC_CLASSDESC, but got 0x%x", buff[0])
		}
		if _, err := reader.Read(buff[:1]); err != nil {
			return err
		}
	}
	//iterate the classes
	for i := len(jo.Classes) - 1; i >= 0; i -= 1 {
		//由于序列化时先序列化父类的Field, 所以要先从父类的Field反序列化
		cc := jo.Classes[i]
		if cc.ScFlag == SC_RW_OBJECT {
			if sub, err := DeserializeScRwObject(reader, refs, cc.ClassName); err != nil {
				return err
			} else {
				cc.RwDatas = []interface{}{sub}
			}
		} else if cc.ScFlag == SC_SERIALIZABLE {
			for _, jf := range cc.Fields {
				if err = ReadJavaField(jf, reader, refs); err != nil {
					return err
				}
			}
		} else {
			StdLogger.Error("[JavaTcObject] Unexpected SC_FLAG [0x%x] for class [%s]", cc.ScFlag, cc.ClassName)
		}
	}

	class0 := jo.Classes[0]
	switch class0.SerialVersionUID {
	case SID_BYTE, SID_SHORT, SID_BOOLEAN, SID_CHARACTER, SID_INTEGER, SID_LONG, SID_FLOAT, SID_DOUBLE:
		jf0 := class0.Fields[0]
		if jf0.FieldName != "value" {
			return fmt.Errorf("8 base type Object, field name should be value, but %s", jf0.FieldName)
		}
		jo.JsonData = jf0.FieldValue
		return nil
	}
	//otherwise is general object

	jsonDatas := make(map[string]interface{})
	jo.JsonData = jsonDatas
	for i, clazz := range jo.Classes {
		jsonDatas[fmt.Sprintf("__class__%d", len(jo.Classes)-i-1)] = clazz.ClassName
		if clazz.ScFlag == SC_RW_OBJECT {
			rwVal := clazz.RwDatas[0]
			if js, ok := rwVal.(JavaSerializer); ok {
				jsVal := js.JsonMap()
				if mp, ok := jsVal.(map[string]interface{}); ok {
					for k, v := range mp {
						jsonDatas[k] = v
					}
				}
			} else {
				StdLogger.Warn("[JavaTcObject] Deserialize Expect JavaSerializer for SC_RW_OBJECT,but got %v\n", rwVal)
			}
			continue
		}
		for _, jf := range clazz.Fields {
			fv := jf.FieldValue
			if js, ok := fv.(JavaSerializer); ok {
				jsonDatas[jf.FieldName] = js.JsonMap()
			} else {
				jsonDatas[jf.FieldName] = fv
			}
		}
	}

	return nil
}

//JsonMap return jsonmap
func (jo *JavaTcObject) JsonMap() interface{} {
	return jo.JsonData
}

//Serialize serialize JavaTcObject to stream
func (jo *JavaTcObject) Serialize(writer io.Writer, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcObject] Serialize >> \n")
	defer StdLogger.Debug("[JavaTcObject] Serialize << \n")

	//make sure the SerialVersionUID
	if jo.SerialVersionUID == 0 {
		jo.SerialVersionUID = jo.Classes[0].SerialVersionUID
	}

	//judge if it's data is null 当前做不到 : (

	buff := make([]byte, 8)
	var err error
	var refIndex int = -1
	//first judge if there is TC_REF already
	for i := 0; i < len(refs); i += 1 {
		var ref interface{} = refs[i]
		if refs[i] == nil {
			break
		}
		if jot, ok := ref.(*JavaTcObject); !ok {
			continue
		} else if jot.SerialVersionUID != jo.SerialVersionUID {
			continue
		} else {
			//judge the json
			if this0, err := json.Marshal(jo); err != nil {
				return err
			} else if this1, err := json.Marshal(jot); err != nil {
				return err
			} else if bytes.Equal(this0, this1) {
				refIndex = i
				break
			}
		}
	}

	if refIndex >= 0 {
		buff[0] = TC_REFERENCE
		refTotal := INTBASE_WIRE_HANDLE + uint32(refIndex)
		binary.BigEndian.PutUint32(buff[1:5], refTotal)
		StdLogger.Debug("[JavaTcObject] Serialize got Reference %d \n", refIndex)
		_, err = writer.Write(buff[:5])
		return err
	}

	//没有ref，开始写TcClassDesc one by one
	buff[0] = TC_OBJECT
	if _, err = writer.Write(buff[:1]); err != nil {
		return err
	}

	for _, cs := range jo.Classes {
		if err = cs.Serialize(writer, refs); err != nil {
			return err
		}
	}
	//类，包括父类写完后 write TC_NULL
	buff[0] = TC_NULL
	if _, err = writer.Write(buff[:1]); err != nil {
		return err
	}
	//add reference
	AddReference(refs, TC_OBJECT, jo)
	// classDesc 有多个，注意每一层classdesc要区分SC_FLAG, 只针对 SC_RW_OBJECT的调用个性化的
	for i := len(jo.Classes) - 1; i >= 0; i -= 1 {
		cc := jo.Classes[i]
		//if SC_FLAG equals 0x03, we invoke the custom serializer
		if cc.ScFlag == SC_RW_OBJECT {
			if err = SerializeScRwObject(writer, refs, cc); err != nil {
				return err
			}
		} else {
			for j, jf := range cc.Fields {
				StdLogger.Debug("[JavaTcObject] Serialize JavaField (%d,%d) %s \n", i, j, jf.FieldName)
				if err = SerializeJavaField(jf, writer, refs); err != nil {
					return err
				}

			}
		}
	}

	return err
}

//SerializeJavaField 注意是序列化它的值，而不是描述符
func SerializeJavaField(jf *JavaField, writer io.Writer, refs []*JavaReferenceObject) error {
	buff := make([]byte, 8)
	var err error
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[SerializeJavaField] %s >> \n", jf.FieldName)
	defer StdLogger.Debug("[SerializeJavaField] %s << \n", jf.FieldName)
	v := jf.FieldValue
	switch jf.FieldType {
	case TC_PRIM_BYTE:
		if b, ok := v.(byte); !ok {
			return fmt.Errorf("Expect byte for TC_PRIM_BYTE, but got %v", v)
		} else {
			buff[0] = b
			if _, err = writer.Write(buff[:1]); err != nil {
				return err
			}
		}
	case TC_PRIM_BOOLEAN:
		if b, ok := v.(bool); !ok {
			return fmt.Errorf("Expect bool for TC_PRIM_BOOLEAN, but got %v", v)
		} else {
			if b {
				buff[0] = 1
			} else {
				buff[0] = 0
			}
			if _, err = writer.Write(buff[:1]); err != nil {
				return err
			}
		}
	case TC_PRIM_CHAR:
		if r, ok := v.(rune); !ok {
			return fmt.Errorf("Expect rune for TC_PRIM_CHAR, but got %v", v)
		} else {
			i16 := (uint16)(r)
			binary.BigEndian.PutUint16(buff[:2], i16)
			if _, err = writer.Write(buff[:2]); err != nil {
				return err
			}
		}
	case TC_PRIM_SHORT:
		if i16, ok := v.(uint16); !ok {
			return fmt.Errorf("Expect short for TC_PRIM_SHORT, but got %v", v)
		} else {
			binary.BigEndian.PutUint16(buff[:2], i16)
			if _, err = writer.Write(buff[:2]); err != nil {
				return err
			}
		}
	case TC_PRIM_INTEGER:
		if i32, ok := v.(uint32); !ok {
			if ii, ok := v.(int); !ok {
				return fmt.Errorf("Expect integer for TC_PRIM_INTEGER, but got %v", v)
			} else {
				binary.BigEndian.PutUint32(buff[:4], uint32(ii))
				if _, err = writer.Write(buff[:4]); err != nil {
					return err
				}
			}
		} else {
			binary.BigEndian.PutUint32(buff[:4], i32)
			if _, err = writer.Write(buff[:4]); err != nil {
				return err
			}
		}
	case TC_PRIM_LONG:
		if i64, ok := v.(uint64); !ok {
			return fmt.Errorf("Expect long for TC_PRIM_LONG, but got %v", v)
		} else {
			binary.BigEndian.PutUint64(buff[:8], i64)
			if _, err = writer.Write(buff[:8]); err != nil {
				return err
			}
		}
	case TC_PRIM_FLOAT:
		if f32, ok := v.(float32); !ok {
			return fmt.Errorf("Expect float for TC_PRIM_FLOAT, but got %v", v)
		} else {
			i32 := math.Float32bits(f32)
			binary.BigEndian.PutUint32(buff[:4], i32)
			if _, err = writer.Write(buff[:4]); err != nil {
				return err
			}
		}
	case TC_PRIM_DOUBLE:
		if f64, ok := v.(float64); !ok {
			return fmt.Errorf("Expect double for TC_PRIM_DOUBLE, but got %v", v)
		} else {
			i64 := math.Float64bits(f64)
			binary.BigEndian.PutUint64(buff[:8], i64)
			if _, err = writer.Write(buff[:8]); err != nil {
				return err
			}
		}
	case TC_OBJ_OBJECT:
		if tco, ok := v.(*JavaTcObject); !ok {
			if tstr, ok := v.(*JavaTcString); ok {
				if err = tstr.Serialize(writer, refs); err != nil {
					return err
				}
			} else if str, ok := v.(string); ok {
				tstr = NewJavaTcString(str)
				if err = tstr.Serialize(writer, refs); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Expect JavaTcObject for TC_OBJ_OBJECT, but got %v", v)
			}
		} else {
			if err = tco.Serialize(writer, refs); err != nil {
				return err
			}
		}
	case TC_OBJ_ARRAY:
		if tArr, ok := v.(*JavaTcArray); !ok {
			return fmt.Errorf("Expect JavaTcArray for TC_OBJ_ARRAY, but got %v", v)
		} else {
			if err = tArr.Serialize(writer, refs); err != nil {
				return err
			}
		}
	}
	return err
}

//Deserialize JavaTcArray deserialize
//davidwang2006@aliyun.com
//2018-01-31 16:37:30
func (tcArr *JavaTcArray) Deserialize(reader io.Reader, refs []*JavaReferenceObject) error {
	//TC_ARRAY开头
	//兼容这个TC_ARRAY是否被消费掉
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcArray] >> ++BEGIN\n")
	defer StdLogger.Debug("[JavaTcArray] << --END\n")
	//firstly, analysis all tc_classdesc
	//TC_OBJECT
	var buff = make([]byte, 4)
	var err error
	if _, err = reader.Read(buff[:1]); err != nil {
		return err
	}
	if TC_REFERENCE == buff[0] { //表示引用了另一个CLASSDESC
		//读剩下的3个字节，后二个字节表示refIndex
		if _, err = reader.Read(buff[:4]); err != nil {
			return err
		} else {
			refIndex := binary.BigEndian.Uint16(buff[2:4])
			ref := refs[int(refIndex)]
			if ref.RefType != TC_ARRAY {
				return fmt.Errorf("[JavaTcArray] Expect ref [%d] type TC_OBJECT, but 0x%x", refIndex, ref.RefType)
			} else if jarrp, ok := ref.Val.(*JavaTcArray); !ok {
				return fmt.Errorf("[JavaTcArray] Expect ref[%d] val *JavaTcObject, but %v", refIndex, ref.Val)
			} else {
				tcArr.ClassDesc = jarrp.ClassDesc
				tcArr.SerialVersionUID = jarrp.SerialVersionUID
				tcArr.Values = jarrp.Values
				return nil
			}
		}
	} else if TC_ARRAY == buff[0] { //证明开头的tc_array未被消费，则再读下一个
		if _, err = reader.Read(buff[:1]); err != nil {
			return err
		}
	}

	//now begin tc_classdesc
	//TC_ARRAY只有一个TC_CLASSDESC
	tcArr.ClassDesc = &JavaTcClassDesc{}
	if err = tcArr.ClassDesc.Deserialize(reader, refs); err != nil {
		return err
	}
	tcArr.SerialVersionUID = tcArr.ClassDesc.SerialVersionUID
	//TC_ARRAY newHandle should added
	AddReference(refs, TC_ARRAY, tcArr)
	//Next should be TC_NULL 0x70
	if b, err := ReadNextByte(reader); err != nil {
		return err
	} else if b != TC_NULL {
		return fmt.Errorf("Expect TC_NULL after TC_CLASSDESC in TC_ARRAY header, but got 0x%x", b)
	}

	var elementCount int
	if b, err := ReadUint32(reader); err != nil {
		return err
	} else {
		StdLogger.Debug("[JavaTcArray] [%s] has %d elements\n", tcArr.ClassDesc.ClassName, b)
		elementCount = int(b)
	}

	//分析className的第二个字节，看是8大基本类型还是'L'
	//String的话 className is [Ljava.lang.String;
	classNameArr := ([]byte)(tcArr.ClassDesc.ClassName)
	eleType := (classNameArr)[1]

	tcArr.Values = make([]interface{}, 0, elementCount)
	tcArr.JsonData = make([]interface{}, 0, elementCount)

	for i := 0; i < elementCount; i++ {
		switch eleType {
		case TC_PRIM_BYTE:
			if b, err := ReadNextByte(reader); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, b)
				tcArr.JsonData = append(tcArr.JsonData, b)
			}
		case TC_PRIM_BOOLEAN:
			if b, err := ReadNextByte(reader); err != nil {
				return err
			} else {
				bv := b == 1
				tcArr.Values = append(tcArr.Values, bv)
				tcArr.JsonData = append(tcArr.JsonData, b)
			}
		case TC_PRIM_CHAR:
			if b, err := ReadUint16(reader); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, rune(b))
				tcArr.JsonData = append(tcArr.JsonData, rune(b))
			}
		case TC_PRIM_SHORT:
			if b, err := ReadUint16(reader); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, b)
				tcArr.JsonData = append(tcArr.JsonData, b)
			}
		case TC_PRIM_INTEGER:
			if b, err := ReadUint32(reader); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, b)
				tcArr.JsonData = append(tcArr.JsonData, b)
			}
		case TC_PRIM_LONG:
			if b, err := ReadUint64(reader); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, b)
				tcArr.JsonData = append(tcArr.JsonData, b)
			}
		case TC_PRIM_FLOAT:
			if b, err := ReadUint32(reader); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, float32(b))
				tcArr.JsonData = append(tcArr.JsonData, float32(b))
			}
		case TC_PRIM_DOUBLE:
			if b, err := ReadUint64(reader); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, float64(b))
				tcArr.JsonData = append(tcArr.JsonData, float64(b))
			}
		case TC_OBJ_ARRAY: //也有可能是数组
			StdLogger.Debug("[JavaTcArray] element[%d] is array too\n", i)
			subArray := &JavaTcArray{}
			if err := subArray.Deserialize(reader, refs); err != nil {
				return err
			} else {
				tcArr.Values = append(tcArr.Values, subArray)
				tcArr.JsonData = append(tcArr.JsonData, subArray.JsonData)
			}
		case TC_OBJ_OBJECT:
			elementClassName := string(classNameArr[2 : len(classNameArr)-1])
			StdLogger.Debug("[JavaTcArray] element[%d] className is %s\n", i, elementClassName)
			if elementClassName == "java.lang.String" {
				if str, err := ReadNextTcString(reader, refs); err != nil {
					return err
				} else {
					tcArr.Values = append(tcArr.Values, str)
					tcArr.JsonData = append(tcArr.JsonData, str)
				}
			} else {
				jo := &JavaTcObject{}
				if err := jo.Deserialize(reader, refs); err != nil {
					return err
				} else {
					tcArr.Values = append(tcArr.Values, jo)
					tcArr.JsonData = append(tcArr.JsonData, jo.JsonData)
				}
			}
		default:
			StdLogger.Error("[JavaTcArray] element[%d] unexpected type %v\n", i, eleType)
		}

	}

	return nil

}

//JsonMap return jsonmap
func (tcArr *JavaTcArray) JsonMap() interface{} {
	return tcArr.JsonData
}

//Serialize JavaTcArray serialize it out to stream
func (tcArr *JavaTcArray) Serialize(writer io.Writer, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaTcArray] Serialize >> \n")
	defer StdLogger.Debug("[JavaTcArray] Serialize << \n")

	//make sure the SerialVersionUID
	if tcArr.SerialVersionUID == 0 {
		tcArr.SerialVersionUID = tcArr.ClassDesc.SerialVersionUID
	}

	//judge if it's data is null 当前做不到 : (

	buff := make([]byte, 8)
	var err error
	var refIndex int = -1
	//first judge if there is TC_REF already
	for i := 0; i < len(refs); i += 1 {
		var ref interface{} = refs[i]
		if refs[i] == nil {
			break
		}
		if refs[i].RefType != TC_ARRAY {
			continue
		} else if jArrP, ok := ref.(*JavaTcArray); !ok {
			continue
		} else if jArrP.SerialVersionUID != tcArr.SerialVersionUID {
			continue
		} else {
			//judge the json
			if this0, err := json.Marshal(tcArr); err != nil {
				return err
			} else if this1, err := json.Marshal(jArrP); err != nil {
				return err
			} else if bytes.Equal(this0, this1) {
				refIndex = i
				break
			}
		}
	}

	if refIndex >= 0 {
		buff[0] = TC_REFERENCE
		refTotal := INTBASE_WIRE_HANDLE + uint32(refIndex)
		binary.BigEndian.PutUint32(buff[1:5], refTotal)
		StdLogger.Debug("[JavaTcArray] Serialize got Reference %d \n", refIndex)
		_, err = writer.Write(buff[:5])
		return err
	}

	//没有ref，开始写TcObject
	buff[0] = TC_ARRAY
	if _, err = writer.Write(buff[:1]); err != nil { // TC_ARRAY
		return err
	}
	//TC_CLASSDESC
	if err = tcArr.ClassDesc.Serialize(writer, refs); err != nil {
		return err
	}
	//类，写完后 write TC_NULL
	buff[0] = TC_NULL
	if _, err = writer.Write(buff[:1]); err != nil {
		return err
	}
	//add reference
	AddReference(refs, TC_ARRAY, tcArr)
	//read elements count
	eleCount := len(tcArr.Values)
	binary.BigEndian.PutUint32(buff[:4], uint32(eleCount))
	//write the length of contents
	if _, err = writer.Write(buff[:4]); err != nil {
		return err
	}
	var ev interface{}
	for i := 0; i < eleCount; i++ {
		ev = tcArr.Values[i]
		rv := reflect.ValueOf(ev)
		var rvType string
		if rv.Kind() == reflect.Ptr {
			rvType = rv.Elem().Type().Name()
		} else {
			rvType = rv.Type().Name()
		}
		StdLogger.Debug("[JavaTcArray] Serialize eles[%d][type=%s] %v >> \n", i, rvType, ev)
		if it, ok := ev.(int); ok {
			binary.BigEndian.PutUint32(buff[:4], uint32(it))
			if _, err = writer.Write(buff[:4]); err != nil {
				return err
			}
		} else if i8, ok := ev.(uint8); ok {
			buff[0] = i8
			if _, err = writer.Write(buff[:1]); err != nil {
				return err
			}
		} else if b, ok := ev.(byte); ok {
			buff[0] = b
			if _, err = writer.Write(buff[:1]); err != nil {
				return err
			}
		} else if i16, ok := ev.(uint16); ok {
			binary.BigEndian.PutUint16(buff[:2], i16)
			if _, err = writer.Write(buff[:2]); err != nil {
				return err
			}
		} else if i32, ok := ev.(uint32); ok {
			binary.BigEndian.PutUint32(buff[:4], i32)
			if _, err = writer.Write(buff[:4]); err != nil {
				return err
			}
		} else if i64, ok := ev.(uint64); ok {
			binary.BigEndian.PutUint64(buff[:8], i64)
			if _, err = writer.Write(buff[:8]); err != nil {
				return err
			}
		} else if f32, ok := ev.(float32); ok {
			i32 := math.Float32bits(f32)
			binary.BigEndian.PutUint32(buff[:4], i32)
			if _, err = writer.Write(buff[:4]); err != nil {
				return err
			}
		} else if f64, ok := ev.(float64); ok {
			i64 := math.Float64bits(f64)
			binary.BigEndian.PutUint64(buff[:8], i64)
			if _, err = writer.Write(buff[:8]); err != nil {
				return err
			}
		} else if str, ok := ev.(string); ok {
			var tcStr = new(JavaTcString)
			*tcStr = (JavaTcString)(str)
			if err = tcStr.Serialize(writer, refs); err != nil {
				return err
			}
		} else if tcStr, ok := ev.(*JavaTcString); ok {
			if err = tcStr.Serialize(writer, refs); err != nil {
				return err
			}
		} else if tcObj_, ok := ev.(*JavaTcObject); ok {
			if err = tcObj_.Serialize(writer, refs); err != nil {
				return err
			}
		} else if tcArr_, ok := ev.(*JavaTcArray); ok {
			if err = tcArr_.Serialize(writer, refs); err != nil {
				return err
			}
		} else {
			StdLogger.Error("[JavaTcArray] Serialize unexpected eles[%d][type=%s] %v >> \n", i, rvType, ev)
			return fmt.Errorf("[JavaTcArray] Serialize unexpected eles[%d][type=%s] %v >> \n", i, rvType, ev)
		}
	}

	return nil
}

//AddClassDesc add java tc class desc to javatcobject
func (jo *JavaTcObject) AddClassDesc(jcd *JavaTcClassDesc) {
	jo.Classes = append(jo.Classes, jcd)
}
