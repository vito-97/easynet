package easynet

import "math"

// abortIndex 表示终止函数调用的下标位置
const abortIndex int8 = math.MaxInt8 >> 1

type IRequest interface {
	GetConnection() IConnection
	GetTcpConnection() IConnection

	GetData() []byte
	GetMsgType() uint32

	GetMessage() IMessage

	//SetHandler 绑定该请求需要执行的所有函数
	SetHandler(handlers HandlersChain)

	//Next 进入下一个方法
	Next()
	//Abort 终止运行
	Abort()
	//IsAborted 是否为终止
	IsAborted() bool
}

type Request struct {
	index      int8
	handlers   HandlersChain
	connection IConnection
	message    IMessage
}

func (r *Request) GetConnection() IConnection {
	return r.connection
}

func (r *Request) GetTcpConnection() IConnection {
	return r.connection
}

func (r *Request) GetData() []byte {
	return r.message.GetData()
}

func (r *Request) GetMsgType() uint32 {
	return r.message.GetType()
}

func (r *Request) GetMessage() IMessage {
	return r.message
}

func (r *Request) SetHandler(handlers HandlersChain) {
	r.handlers = handlers
}

func (r *Request) Next() {
	for r.index < int8(len(r.handlers)) {
		handler := r.handlers[r.index]
		r.index++
		handler(r)
	}
}

func (r *Request) Abort() {
	r.index = abortIndex
}

func (r *Request) IsAborted() bool {
	return r.index >= abortIndex
}

func NewRequest(connection IConnection, message IMessage) IRequest {
	return &Request{
		connection: connection,
		message:    message,
	}
}
