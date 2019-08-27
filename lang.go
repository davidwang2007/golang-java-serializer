package main

import "io"
import "encoding/binary"
import "fmt"

//定义基础类型
//author: davidwang2006@aliyun.com
//date: 2018-01-29 15:32:15

const (
	TC_NULL          byte = 0x70 | iota //0x70
	TC_REFERENCE     byte = 0x70 | iota //0x71
	TC_CLASSDESC     byte = 0x70 | iota //0x72
	TC_OBJECT        byte = 0x70 | iota //0x73
	TC_STRING        byte = 0x70 | iota //0x74
	TC_ARRAY         byte = 0x70 | iota //0x75
	TC_CLASS         byte = 0x70 | iota //0x76
	TC_BLOCKDATA     byte = 0x70 | iota //0x77
	TC_ENDBLOCKDATA  byte = 0x70 | iota //0x78
	TC_RESET         byte = 0x70 | iota //0x79
	TC_BLOCKDATALONG byte = 0x70 | iota //0x7A
	TC_EXCEPTION     byte = 0x70 | iota //0x7B
)

//type code define
//用来描述field类型的
const (
	TC_PRIM_BYTE    byte = 'B'
	TC_PRIM_CHAR    byte = 'C'
	TC_PRIM_DOUBLE  byte = 'D'
	TC_PRIM_FLOAT   byte = 'F'
	TC_PRIM_INTEGER byte = 'I'
	TC_PRIM_LONG    byte = 'J'
	TC_PRIM_SHORT   byte = 'S'
	TC_PRIM_BOOLEAN byte = 'Z'
	TC_OBJ_ARRAY    byte = '['
	TC_OBJ_OBJECT   byte = 'L'
)

//define types mapping to java types
type JavaShort int16
type JavaInt int32
type JavaLong int64

//unsigned
type JavaUShort uint16
type JavaUInt uint32
type JavaULong uint64

//float types
type JavaFloat float32
type JavaDouble float64

//some stream indicators
const STREAM_MAGIC = 0xACED
const STREAM_VERSION = 0x0005
const INTBASE_WIRE_HANDLE = 0x007E0000

//class desc flags, serializable flag
const SC_WRITE_METHOD byte = 0x01
const SC_SERIALIZABLE byte = 0x02 //only support this one
const SC_RW_OBJECT byte = 0x03    //拥有自己的writeObject, readObject, for example: HashMap, 此种类型需要每一个定义一个相应的结构体
const SC_EXTERNALIZABLE byte = 0x04

//define some serialiable objects' serialVersionUID
const (
	SID_STRING_ARRAY uint64 = 0xADD256E7E91D7B47
	SID_BYTE_ARRAY   uint64 = 0xACF317F8060854E0
	SID_INT_ARRAY    uint64 = 0x4DBA602676EAB2A5
	SID_SHORT_ARRAY  uint64 = 0xEF832E06E55DB0FA
	SID_LONG_ARRAY   uint64 = 0x782004B512B17593
	SID_INTEGER      uint64 = 1360826667806852920 //decimal
	SID_LONG         uint64 = 4290774380558885855 //decimal
	SID_SHORT        uint64 = 7515723908773894738 //decimal
	SID_BYTE         uint64 = 0x9C4E6084EE50F51C
	SID_FLOAT        uint64 = 0xDAEDC9A2DB3CF0EC
	SID_DOUBLE       uint64 = 0x80B3C24A296BFB04
	SID_BOOLEAN      uint64 = 0xCD207280D59CFAEE
	SID_CHARACTER    uint64 = 3786198910865385080 //decimal
)

//JavaReferenceObject java reference object
type JavaReferenceObject struct {
	RefType byte        //引用类型，TC_OBJECT, TC_CLASSDESC, TC_ARRAY, TC_STRING 共4种
	Val     interface{} //引用的值
}

type JavaSerializer interface {
	Serialize(io.Writer, []*JavaReferenceObject) error
	Deserialize(io.Reader, []*JavaReferenceObject) error
	JsonMap() interface{}
}

//IsPrimType judge if target prim type is 8 base prim type
func IsPrimType(typ byte) bool {
	switch typ {
	case TC_PRIM_BYTE, TC_PRIM_CHAR, TC_PRIM_BOOLEAN, TC_PRIM_INTEGER, TC_PRIM_SHORT, TC_PRIM_FLOAT, TC_PRIM_DOUBLE, TC_PRIM_LONG:
		return true
	default:
		return false
	}
}

//NewJavaReferencePool
//new Java Reference Pool to hold the TC_REF Object
func NewJavaReferencePool(poolSize int) []*JavaReferenceObject {
	if poolSize < 10 {
		poolSize = 1 << 1 //1024
	}
	refPool := make([]*JavaReferenceObject, poolSize) //128
	return refPool
}

//ReadNextJavaField read next java field desc
func ReadNextJavaField(reader io.Reader, refs []*JavaReferenceObject) (*JavaField, error) {
	var jf = &JavaField{}
	var err error
	if jf.FieldType, err = ReadNextByte(reader); err != nil {
		return nil, err
	}
	//field name length
	var fNameLen uint16
	if fNameLen, err = ReadUint16(reader); err != nil {
		return nil, err
	}

	if jf.FieldName, err = ReadUTFString(reader, int(fNameLen)); err != nil {
		return nil, err
	}
	switch jf.FieldType {
	case TC_OBJ_OBJECT:
		fallthrough
	case TC_OBJ_ARRAY:
		//跟着的为TC_STRING
		if jf.FieldObjectClassName, err = ReadNextTcString(reader, refs); err != nil {
			return nil, err
		}
	}

	return jf, nil
}

//AddReference add java reference object
func AddReference(refs []*JavaReferenceObject, refType byte, refVal interface{}) {
	StdLogger.LevelUp()
	defer StdLogger.LevelDown()
	var i int
	for i = 0; i < cap(refs); i++ {
		if refs[i] == nil {
			refs[i] = &JavaReferenceObject{
				refType,
				refVal,
			}
			StdLogger.Debug("[REFERENCE] [ADD] [%d] refType:0x%x, refVal:%v\n", i, refType, refVal)
			return
		}
	}
	StdLogger.Error("[REFERENCE] [ADD] There is no enough room for JavaReferenceObject, current cap %d\n", cap(refs))
}

//ReadUint16 read uint16, aka java short
func ReadUint16(reader io.Reader) (uint16, error) {
	if bs, err := ReadNextBytes(reader, 2); err != nil {
		return 0, err
	} else {
		return binary.BigEndian.Uint16(bs), nil
	}
}

//ReadUint32 read uint16, aka java int
func ReadUint32(reader io.Reader) (uint32, error) {
	if bs, err := ReadNextBytes(reader, 4); err != nil {
		return 0, err
	} else {
		return binary.BigEndian.Uint32(bs), nil
	}
}

//ReadUint64 read uint16, aka java int
func ReadUint64(reader io.Reader) (uint64, error) {
	if bs, err := ReadNextBytes(reader, 8); err != nil {
		return 0, err
	} else {
		return binary.BigEndian.Uint64(bs), nil
	}
}

//ReadUTFString read utf8 string from the input stream
func ReadUTFString(reader io.Reader, len int) (string, error) {
	if bs, err := ReadNextBytes(reader, len); err != nil {
		return "", err
	} else {
		return string(bs), nil
	}

}

//ReadNextBytes read next bytes from the stream
func ReadNextBytes(reader io.Reader, n int) ([]byte, error) {
	bs := make([]byte, n)
	if c, err := reader.Read(bs); err != nil {
		return nil, err
	} else if c != n {
		return nil, fmt.Errorf("Try to read %d bytes, but got %d bytes", n, c)
	}
	return bs, nil
}

//ReadNextByte read next byte from the stream
func ReadNextByte(reader io.Reader) (byte, error) {
	var n = 1
	bs := make([]byte, n)
	if c, err := reader.Read(bs); err != nil {
		return 0, err
	} else if c != n {
		return 0, fmt.Errorf("Try to read %d bytes, but got %d bytes", n, c)
	}
	return bs[0], nil
}

//ReadNextTcString read next tc string
//TC_STRING + string length + string
func ReadNextTcString(reader io.Reader, refs []*JavaReferenceObject) (string, error) {
	if b, err := ReadNextByte(reader); err != nil {
		return "", err
	} else if b == TC_REFERENCE {
		//to be continued...
		if refIndex, err := ReadUint32(reader); err != nil {
			return "", err
		} else {
			var ref = refs[refIndex-0x007E0000]
			if v, ok := ref.Val.(string); !ok {
				return "", fmt.Errorf("Expected string, but got %v", ref.Val)
			} else {
				return v, nil
			}
		}
	} else if b == TC_NULL { //考虑String为null的情况
		return "", nil
	} else if b != TC_STRING {
		return "", fmt.Errorf("Expected 0x%x, but got 0x%x", TC_STRING, b)
	}

	if strLen, err := ReadUint16(reader); err != nil {
		return "", err
	} else if str, err := ReadUTFString(reader, int(strLen)); err != nil {
		return "", err
	} else {
		//如果读出来原始的TC_STRING则产生一个新的 newHandle
		AddReference(refs, TC_STRING, str)
		return str, nil
	}
}

//DeserializeStream
//deserialize stream to java object
func DeserializeStream(reader io.Reader) (JavaSerializer, error) {
	//read magic
	if b, err := ReadUint16(reader); err != nil {
		return nil, err
	} else if b != uint16(STREAM_MAGIC) {
		return nil, fmt.Errorf("stream should start with STREAM_MAGIC but got 0x%x", b)
	}
	//read version
	if b, err := ReadUint16(reader); err != nil {
		return nil, err
	} else if b != uint16(STREAM_VERSION) {
		return nil, fmt.Errorf("stream should start with STREAM_VERSION but got 0x%x", b)
	}

	refs := NewJavaReferencePool(1 << 10) //make([]*JavaReferenceObject, 1000)

	if b, err := ReadNextByte(reader); err != nil {
		return nil, err
	} else {
		switch b {
		case TC_ARRAY:
			tcArr := &JavaTcArray{}
			if err = tcArr.Deserialize(reader, refs); err != nil {
				return nil, err
			} else {
				return tcArr, nil
			}
		case TC_OBJECT:
			tcJo := &JavaTcObject{}
			if err = tcJo.Deserialize(reader, refs); err != nil {
				return nil, err
			} else {
				return tcJo, nil
			}
		case TC_STRING:
			tcStr := new(JavaTcString)
			if err = tcStr.Deserialize(reader, refs); err != nil {
				return nil, err
			} else {
				return tcStr, nil
			}
		case TC_NULL: //表示空指针
			StdLogger.Warn("Stream's body first byte is TC_NULL")
			return new(JavaTcString), nil
		default:
			return nil, fmt.Errorf("stream should be one of TC_ARRAY & TC_OBJECT & TC_STRING, but got 0x%x", b)
		}
	}

}

//SerializeJavaEntity
//serialize java entity to stream
func SerializeJavaEntity(writer io.Writer, entity JavaSerializer) error {
	var err error
	buff := make([]byte, 4)
	binary.BigEndian.PutUint16(buff[:2], STREAM_MAGIC)
	binary.BigEndian.PutUint16(buff[2:4], STREAM_VERSION)
	if _, err = writer.Write(buff); err != nil {
		return err
	}
	refs := NewJavaReferencePool(1 << 7)

	return entity.Serialize(writer, refs)
}
