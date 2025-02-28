package easynet

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type IDataPack interface {
	HeadLength() uint32
	Pack(msg IMessage) ([]byte, error)
	Unpack(b []byte) (IMessage, error)
}

var defaultHeadLength uint32 = TLVHeaderSize

type DataPack struct{}

func (d *DataPack) Pack(msg IMessage) ([]byte, error) {
	buffer := bytes.NewBuffer([]byte{})
	//写入消息类型
	if err := binary.Write(buffer, binary.BigEndian, msg.Type()); err != nil {
		return nil, err
	}
	//写入消息长度
	if err := binary.Write(buffer, binary.BigEndian, msg.Length()); err != nil {
		return nil, err
	}
	//写入消息内容
	if err := binary.Write(buffer, binary.BigEndian, msg.Data()); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (d *DataPack) Unpack(b []byte) (IMessage, error) {
	reader := bytes.NewReader(b)

	msg := &Message{}

	//读取数据类型
	if err := binary.Read(reader, binary.BigEndian, &msg.t); err != nil {
		return nil, err
	}

	//读取数据长度
	if err := binary.Read(reader, binary.BigEndian, &msg.len); err != nil {
		return nil, err
	}

	//判断数据的长度是否为允许的最大包长度
	if GlobalConfig.MaxPacketSize > 0 && msg.Length() > GlobalConfig.MaxPacketSize {
		return nil, errors.New("too large msg data received")
	}

	return msg, nil
}

func (d *DataPack) HeadLength() uint32 {
	return defaultHeadLength
}

func NewDataPack() IDataPack {
	return &DataPack{}
}
