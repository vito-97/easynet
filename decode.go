package easynet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

const TLVTypeSize = 4
const TLVLengthSize = 4
const TLVHeaderSize = TLVTypeSize + TLVLengthSize //表示TLV空包长度

type IDecode interface {
	Handler() HandlerFunc
	Tag() uint32
	Length() uint32
	Value() []byte
	Decode(b []byte) IDecode
	LengthField() *LengthField
}

type TLVDecode struct {
	tag    uint32
	length uint32
	value  []byte

	lengthField *LengthField
}

func (t *TLVDecode) Handler() HandlerFunc {
	return func(req IRequest) {
		msg := req.Message()

		if msg == nil {
			return
		}

		data := msg.Data()

		if len(data) < TLVHeaderSize {
			return
		}

		dc := t.Decode(data)
		msg.SetType(dc.Tag())
		msg.SetLength(dc.Length())
		msg.SetData(dc.Value())
	}
}

func (t *TLVDecode) Tag() uint32 {
	return t.tag
}

func (t *TLVDecode) Length() uint32 {
	return t.length
}

func (t *TLVDecode) Value() []byte {
	return t.value
}

func (t *TLVDecode) Decode(b []byte) IDecode {
	d := &TLVDecode{}
	d.tag = binary.BigEndian.Uint32(b[0:4])
	d.length = binary.BigEndian.Uint32(b[4:8])
	d.value = make([]byte, d.length)

	//读取value内容
	binary.Read(bytes.NewBuffer(b[8:8+d.length]), binary.BigEndian, d.value)
	return d
}

func (t *TLVDecode) LengthField() *LengthField {
	return t.lengthField
}

func NewTLVDecoder() IDecode {
	return &TLVDecode{
		lengthField: &LengthField{
			MaxLength:         math.MaxUint32 + TLVHeaderSize,
			LengthFieldOffset: 4,
			LengthFieldLength: 4,
			LengthAdjustment:  0,
			SkipLength:        0,
		},
	}
}

type IFrameDecode interface {
	Decode(buff []byte) [][]byte
	New() IFrameDecode
}

type FrameDecode struct {
	LengthField
	EndOffset             int   //长度字段结束位置的偏移量  LengthFieldOffset+LengthFieldLength
	discardingTooLongData bool  //true 表示开启丢弃模式，false 正常工作模式
	tooLongDataLength     int64 //当某个数据包的长度超过maxLength，则开启丢弃模式，此字段记录需要丢弃的数据长度
	discardLength         int64 //记录还剩余多少字节需要丢弃
	in                    []byte
	lock                  sync.Mutex
}

func (d *FrameDecode) New() IFrameDecode {
	return &FrameDecode{
		LengthField: d.LengthField,
		EndOffset:   d.EndOffset,
		in:          make([]byte, 0),
	}
}

func (d *FrameDecode) Decode(buff []byte) [][]byte {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.in = append(d.in, buff...)
	res := make([][]byte, 0)

	for {
		arr := d.decode()

		if arr != nil {
			res = append(res, arr)
			size := len(arr) + d.SkipLength
			if size > 0 {
				d.in = d.in[size:]
			}
		} else {
			return res
		}
	}
}

// decode 解包
func (d *FrameDecode) decode() []byte {
	in := bytes.NewBuffer(d.in)

	if d.discardingTooLongData {
		d.discardingTooLongDataFn(in)
	}

	//判断缓冲区的可读字节是否少于长度偏移量
	if d.EndOffset > in.Len() {
		//长度还不完整，半包
		return nil
	}

	// 获取数据长度 不包括LengthAdjustment的调整值
	dataLength := d.getUnadjustedDataLength(in)

	if dataLength < 0 {
		//内部会跳过这个数据包的字节数，并抛异常
		d.failOnNegativeLengthField(in, dataLength)
	}

	dataLength += int64(d.EndOffset) + int64(d.LengthAdjustment)

	//如果数据包长度大于最大长度
	if uint64(dataLength) > d.MaxLength {
		//对超出的部分进行处理
		d.exceededDataLength(in, dataLength)
		return nil
	}

	dataLengthInt := int(dataLength)

	//半包
	if in.Len() < dataLengthInt {
		return nil
	}

	//跳过的字节数是否大于数据包长度
	if d.SkipLength > dataLengthInt {
		d.failOnDataLengthLessThanInitialBytesToStrip(in, dataLength)
	}

	//跳过initialBytesToStrip个字节
	in.Next(d.SkipLength)

	//获取跳过后的真实数据长度
	length := dataLengthInt - d.SkipLength
	//提取真实的数据
	buff := make([]byte, length)
	in.Read(buff)
	return buff
}

// 丢弃过长数据
func (d *FrameDecode) discardingTooLongDataFn(buffer *bytes.Buffer) {
	//保存当前还需要丢弃多少字节
	discardLength := d.discardLength
	//当前可以丢弃的字节
	length := math.Min(float64(discardLength), float64(buffer.Len()))
	//丢弃
	buffer.Next(int(length))
	//更新还需要丢弃的字节数
	discardLength -= int64(length)
	d.discardLength = discardLength

	d.failIfNecessary()
}

// 判断是否丢弃完成
func (d *FrameDecode) failIfNecessary() {
	if d.discardLength == 0 {
		//关闭丢弃模式
		d.discardingTooLongData = false
		d.tooLongDataLength = 0
	}
}

// getUnadjustedDataLength 获取未对齐的数据长度
func (d *FrameDecode) getUnadjustedDataLength(buf *bytes.Buffer) int64 {
	var dataLength int64

	length := d.LengthFieldLength
	order := d.Order
	offset := d.LengthFieldOffset
	data := buf.Bytes()
	data = data[offset : offset+length]
	buffer := bytes.NewBuffer(data)
	switch length {
	case 1:
		var value uint8
		binary.Read(buffer, order, &value)
		dataLength = int64(value)
	case 2:
		var value uint16
		binary.Read(buffer, order, &value)
		dataLength = int64(value)
	case 3:
		//通过位运算算出3字节的数据长度
		if order == binary.LittleEndian {
			value := uint(data[0]) | uint(data[1])<<8 | uint(data[2])<<16
			dataLength = int64(value)
		} else {
			value := uint(data[2]) | uint(data[1])<<8 | uint(data[0])<<16
			dataLength = int64(value)
		}
	case 4:
		var value uint32
		binary.Read(buffer, order, &value)
		dataLength = int64(value)
	case 8:
		binary.Read(buffer, order, &dataLength)
	default:
		panic(fmt.Sprintf("unsupported LengthFieldLength: %d (expected: 1, 2, 3, 4, or 8)", d.LengthFieldLength))
	}

	return dataLength
}

// failOnNegativeLengthField 负长度数据时报错
func (d *FrameDecode) failOnNegativeLengthField(buffer *bytes.Buffer, dataLength int64) {
	offset := d.EndOffset
	buffer.Next(offset)
	panic(fmt.Sprintf("negative pre-adjustment length field: %d", dataLength))
}

// exceededDataLength 判断当前缓冲的数据是否足够
func (d *FrameDecode) exceededDataLength(buffer *bytes.Buffer, dataLength int64) {
	discardLength := dataLength - int64(buffer.Len())
	d.tooLongDataLength = dataLength

	//包的长度足够
	if discardLength < 0 {
		//丢弃当前包数据
		buffer.Next(int(dataLength))
	} else {
		//开启丢弃模式
		d.discardingTooLongData = true
		//记录下一次需要丢弃多少字节
		d.discardLength = discardLength
		//丢弃所有数据
		buffer.Next(buffer.Len())
	}

	d.failIfNecessary()
}

// failOnDataLengthLessThanInitialBytesToStrip 跳过数据的长度大于获取到的数据长度
func (d *FrameDecode) failOnDataLengthLessThanInitialBytesToStrip(in *bytes.Buffer, dataLength int64) {
	initialBytesToStrip := d.SkipLength
	in.Next(int(dataLength))
	panic(fmt.Sprintf("adjusted data length (%d) is less than SkipLength: %d", dataLength, initialBytesToStrip))
}

func NewFrameDecode(field LengthField) IFrameDecode {
	d := &FrameDecode{
		LengthField: field,
	}

	if d.Order == nil {
		d.Order = binary.BigEndian
	}

	d.EndOffset = d.LengthFieldOffset + d.LengthFieldLength
	d.in = make([]byte, 0)

	return d
}
