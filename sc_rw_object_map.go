package main

import "io"
import "math"
import "reflect"
import "encoding/binary"
import "fmt"

const SID_HASH_MAP = 362498820763181265
const SID_LINKED_HASH_MAP = 3801124242820219131

//JavaHashMap
type JavaHashMap struct {
	ClassDesc  *JavaTcClassDesc
	LoadFactor float32
	Thredshold uint32
	Buckets    uint32
	Entries    map[string]interface{} //golang json unmashall does not support interface{} type as it's key
}

//GenerateHashMapClassDesc
func GenerateHashMapClassDesc(datas []interface{}) *JavaTcClassDesc {
	jtc := &JavaTcClassDesc{}
	jtc.SerialVersionUID = SID_HASH_MAP
	jtc.ClassName = "java.util.HashMap"
	jtc.ScFlag = SC_RW_OBJECT
	jtc.Fields = GenerateHashMapFields(len(datas))
	jtc.SortFields()
	jtc.RwDatas = datas
	return jtc
}

//GenerateLinkedHashMapClassDesc
func GenerateLinkedHashMapClassDesc() *JavaTcClassDesc {
	jtc := &JavaTcClassDesc{}
	jtc.SerialVersionUID = SID_LINKED_HASH_MAP
	jtc.ClassName = "java.util.LinkedHashMap"
	jtc.ScFlag = SC_SERIALIZABLE
	jf := NewJavaField(TC_PRIM_BOOLEAN, "accessOrder", false) //does not matter, true or false
	jtc.Fields = []*JavaField{jf}
	return jtc
}

//GenerateHashMapFields generate HashMapFields
func GenerateHashMapFields(size int) []*JavaField {
	jfs := make([]*JavaField, 2)
	var loadFactor float32 = 0.75
	jf := &JavaField{
		FieldType:  TC_PRIM_FLOAT,
		FieldName:  "loadFactor",
		FieldValue: loadFactor,
	}
	var threshold uint32 = uint32(size) << 1
	jf2 := &JavaField{
		FieldType:  TC_PRIM_INTEGER,
		FieldName:  "threshold",
		FieldValue: threshold,
	}
	jfs[0] = jf
	jfs[1] = jf2

	return jfs
}

//NewHashMap new hash map
func NewHashMap(mp map[string]interface{}) *JavaTcObject {
	items := MapData2Slice(mp)
	clzDesc := GenerateHashMapClassDesc(items)
	jo := NewJavaTcObject(SID_HASH_MAP)
	jo.AddClassDesc(clzDesc)
	return jo
}

//NewLinkedHashMap new hash map
func NewLinkedHashMap(mp map[string]interface{}) *JavaTcObject {
	items := MapData2Slice(mp)
	clzDesc := GenerateHashMapClassDesc(items)
	jo := NewJavaTcObject(SID_LINKED_HASH_MAP)
	jo.AddClassDesc(GenerateLinkedHashMapClassDesc())
	jo.AddClassDesc(clzDesc)
	return jo
}

//Deserialize 从classdata部分开始读取
func (mp *JavaHashMap) Deserialize(reader io.Reader, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaHashMap] >>\n")
	defer StdLogger.Debug("[JavaHashMap] <<\n")

	//loadFactor
	if lf, err := ReadUint32(reader); err != nil {
		return err
	} else {
		mp.LoadFactor = math.Float32frombits(lf)
	}
	//threshold
	if ts, err := ReadUint32(reader); err != nil {
		return err
	} else {
		mp.Thredshold = ts
	}

	//must be 0x77 TC_BLOCKDATA
	if b, err := ReadNextByte(reader); err != nil {
		return err
	} else if b != TC_BLOCKDATA {
		return fmt.Errorf("There should be TC_BLOCKDATA, but got 0x%x", b)
	}
	//should follow by 0x08, 表示8字节后是所有的Entry
	if b, err := ReadNextByte(reader); err != nil {
		return err
	} else if b != 0x08 {
		return fmt.Errorf("There should be 0x08, but got 0x%x", b)
	}

	if bt, err := ReadUint32(reader); err != nil {
		return err
	} else {
		mp.Buckets = bt
		StdLogger.Debug("[JavaHashMap] has %d buckest\n", bt)
	}
	//size
	var size int
	if sz, err := ReadUint32(reader); err != nil {
		return err
	} else {
		size = int(sz)
	}
	StdLogger.Debug("[JavaHashMap] has %d entries\n", size)
	mp.Entries = make(map[string]interface{})

	for i := 0; i < size; i += 1 {
		StdLogger.Debug("[JavaHashMap] try to read entry [%d]\n", i)
		if k, err := ReadNextEle(reader, refs); err != nil {
			StdLogger.Error("[JavaHashMap] Error when read %d entry's key: %v\n", i, err)
			return err
		} else if v, err := ReadNextEle(reader, refs); err != nil {
			StdLogger.Error("[JavaHashMap] Error when read %d entry's value: %v\n", i, err)
			return err
		} else {
			StdLogger.Debug("[JavaHashMap] Got Entry [%d] %v <-> %v\n", i, k, v)
			mp.Entries[fmt.Sprintf("%v", k.JsonMap())] = v.JsonMap()
		}

	}
	//
	//must be 0x78 TC_ENDBLOCKDATA
	if b, err := ReadNextByte(reader); err != nil {
		return err
	} else if b != TC_ENDBLOCKDATA {
		return fmt.Errorf("There should be TC_ENDBLOCKDATA, but got 0x%x", b)
	}

	return nil
}

//JsonMap return json style data
func (mp *JavaHashMap) JsonMap() interface{} {
	return mp.Entries
}

func (mp *JavaHashMap) Serialize(writer io.Writer, refs []*JavaReferenceObject) error {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	StdLogger.Debug("[JavaHashMap] Serialize >>\n")
	defer StdLogger.Debug("[JavaHashMap] Serialize <<\n")

	buff := make([]byte, 8)
	var err error

	//loadFactor 0.75
	var loadFactor float32 = 0.75
	var ui32 uint32 = math.Float32bits(loadFactor)
	binary.BigEndian.PutUint32(buff[:4], ui32)
	if _, err = writer.Write(buff[:4]); err != nil {
		return err
	}
	//write threhold
	datas := mp.ClassDesc.RwDatas
	ui32 = uint32(len(datas) << 1)
	binary.BigEndian.PutUint32(buff[:4], ui32)
	if _, err = writer.Write(buff[:4]); err != nil {
		return err
	}

	buff[0] = TC_BLOCKDATA
	buff[1] = 0x08
	if _, err = writer.Write(buff[:2]); err != nil {
		return err
	}
	//buckets
	binary.BigEndian.PutUint32(buff[:4], ui32)
	if _, err = writer.Write(buff[:4]); err != nil {
		return err
	}
	//entryies count
	ui32 = uint32(len(datas) / 2)
	binary.BigEndian.PutUint32(buff[:4], ui32)
	if _, err = writer.Write(buff[:4]); err != nil {
		return err
	}

	for i := 0; i < len(datas); i += 1 {
		var item interface{} = datas[i]
		//StdLogger.Warn("Got item %d %v\n", i, item)
		if str, ok := item.(string); ok {
			tcStr := new(JavaTcString)
			*tcStr = (JavaTcString)(str)
			if err = tcStr.Serialize(writer, refs); err != nil {
				return err
			}
		} else if tcStr, ok := item.(*JavaTcString); ok {
			if err = tcStr.Serialize(writer, refs); err != nil {
				return err
			}
		} else if jo, ok := item.(*JavaTcObject); ok {
			if err = jo.Serialize(writer, refs); err != nil {
				return err
			}
		} else if jArr, ok := item.(*JavaTcArray); ok {
			if err = jArr.Serialize(writer, refs); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Unsupport map entry type %v", item)
		}

	}

	buff[0] = TC_ENDBLOCKDATA
	_, err = writer.Write(buff[:1])
	return err
}

//MapData2Slice wrap map to slice
func MapData2Slice(mp map[string]interface{}) []interface{} {

	slade := make([]interface{}, 0, len(mp)*2)
	for k, v := range mp {
		slade = append(slade, k)
		tv := reflect.TypeOf(v)
		if tv.Kind() != reflect.Slice {
			slade = append(slade, v)
		} else {
			te := tv.Elem().Kind()
			switch te {
			case reflect.Uint8: //表示字节流数组
				if tmpArr, ok := v.([]byte); !ok {
					StdLogger.Error("Expect []byte for key %s, but got %v\n", k, v)
					return nil
				} else {
					jArr := NewByteArray(tmpArr)
					slade = append(slade, jArr)
				}
			case reflect.String:
				if tmpArr, ok := v.([]string); !ok {
					StdLogger.Error("Expect []byte for key %s, but got %v\n", k, v)
					return nil
				} else {
					jArr := NewStringArray(tmpArr)
					slade = append(slade, jArr)
				}

			default:
				StdLogger.Error("Unsupport hashmap value type %s for %v\n", te, v)
				return nil
			}
		}

	}

	return slade
}
