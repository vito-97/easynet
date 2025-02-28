package easynet

type IMessage interface {
	Type() uint32
	SetType(t uint32)
	Data() []byte
	SetData(b []byte)
	Length() uint32
	SetLength(len uint32)
}

type Message struct {
	t    uint32
	data []byte
	len  uint32
}

func (m *Message) SetType(t uint32) {
	m.t = t
}

func (m *Message) SetData(b []byte) {
	m.data = b
}

func (m *Message) SetLength(len uint32) {
	m.len = len
}

func (m *Message) Type() uint32 {
	return m.t
}

func (m *Message) Data() []byte {
	return m.data
}

func (m *Message) Length() uint32 {
	return m.len
}

func NewMessage(data []byte) *Message {
	return &Message{
		len:  uint32(len(data)),
		data: data,
	}
}

func NewMessageWithType(t uint32, data []byte) *Message {
	return &Message{
		t:    t,
		len:  uint32(len(data)),
		data: data,
	}
}
