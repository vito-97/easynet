package easynet

import "encoding/binary"

type LengthField struct {
	Order             binary.ByteOrder //大小端
	MaxLength         uint64           //最大长度
	LengthFieldOffset int              //长度字段偏移量
	LengthFieldLength int              //长度域字段的字节数
	LengthAdjustment  int              //长度调整（有些协议长度包括头部）
	SkipLength        int              //需要跳过的字节数
}
