package easynet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type IConnection interface {
	Start()
	Stop()
	IsStopped() bool
	StartReader()
	StartWriter()

	GetID() uint64
	GetName() string

	Send(data []byte) error
	SendToQueue(data []byte) error

	SendMsg(t uint32, data []byte) error
	SendMsgBuff(t uint32, data []byte) error

	SetUid(uid uint64)
	GetUid() uint64

	GetWorker() IWorker
	GetUseWorkerStatus() bool
	GetWorkerId() uint32
	SetWorkerId(id uint32)

	GetConnection() net.Conn
	GetTcpConnection() net.Conn

	GetRemoteAddr() net.Addr
	GetLocalAddr() net.Addr
	GetLocalAddrString() string
	GetRemoteAddrString() string

	SetProperty(key string, value interface{})
	GetProperty(key string) (interface{}, error)
	DeleteProperty(key string) bool

	Content() context.Context
}

type Connection struct {
	Switch
	writerSwitch Switch

	conn net.Conn

	name string
	id   uint64
	uid  uint64

	worker          IWorker
	workerId        uint32
	useWorkerStatus bool

	lock     sync.RWMutex
	property map[string]interface{}

	//数据报文封包方式
	dp IDataPack
	//断黏包解码器
	fd IFrameDecode

	ctx    context.Context
	cancel context.CancelFunc

	onConnStart []ConnCallback
	onConnStop  []ConnCallback

	//消息推送
	msgChan chan []byte
}

func (c *Connection) Start() {
	if c.isStarted() {
		return
	}
	if !c.setStarted() {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			debugPrint("conn start error:%v\n", err)
		}
	}()

	c.callOnConnStart()
	go c.StartReader()
}

func (c *Connection) Stop() {
	if c.isStopped() {
		return
	}
	if !c.setStopped() {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			debugPrint("conn stop error:%v\n", err)
		}
	}()

	c.callOnConnStop()
	c.cancel()

	_ = c.conn.Close()

	debugPrint("conn stop, id = %d\n", c.id)
}

func (c *Connection) IsStopped() bool {
	return c.isStopped()
}

func (c *Connection) StartReader() {
	defer c.Stop()
	defer func() {
		if err := recover(); err != nil {
			debugPrint("conn id=%d panic error=%v\n", c.id, err)
			debugPrint(string(stack(2)))
		}
	}()
	var b = make([]byte, GlobalConfig.IOReadBuffSize)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			//读取连接的IO数据
			n, err := c.conn.Read(b)

			if err != nil && err != io.EOF {
				debugPrint("conn id=%d read error=%v\n", c.id, err)
				return
			}

			buf := b[:n]

			var groups [][]byte

			//处理自定义协议断粘包问题
			if c.fd != nil {
				groups = c.fd.Decode(buf)
			} else {
				groups = [][]byte{
					buf,
				}
			}

			if groups == nil {
				continue
			}

			for _, data := range groups {
				message := NewMessage(data)
				request := NewRequest(c, message)
				c.worker.Execute(request)
			}
		}
	}
}

func (c *Connection) StartWriter() {
	if !c.writerSwitch.isStarted() {
		debugPrint("writer goroutine %d start fail, must be lazy start\n", c.id)
		return
	}
	debugPrint("writer goroutine %d is running\n", c.id)
	defer debugPrint("writer goroutine %d is exit\n", c.id)

	for {
		select {
		case data, ok := <-c.msgChan:
			if !ok {
				debugPrint("writer chan %d is closed\n", c.id)
				break
			}
			err := c.Send(data)
			if err != nil {
				debugPrint("send buff data error:%s writer %d is exited\n", err, c.id)
				break
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Connection) Content() context.Context {
	return c.ctx
}

func (c *Connection) Send(data []byte) error {
	if c.isStopped() {
		return errors.New(fmt.Sprintf("connection %d is closed when send", c.id))
	}

	_, err := c.conn.Write(data)

	if err != nil {
		debugPrint("send msg error, data = [%s], err = %s\n", data, err)
	}

	return err
}

func (c *Connection) getMsgData(t uint32, data []byte) ([]byte, error) {
	message := NewMessageWithType(t, data)
	return c.dp.Pack(message)
}

func (c *Connection) SendMsg(t uint32, data []byte) error {
	if c.isStopped() {
		return errors.New(fmt.Sprintf("connection %d is closed when send msg", c.id))
	}

	b, err := c.getMsgData(t, data)

	if err != nil {
		return err
	}

	return c.Send(b)
}

func (c *Connection) SendToQueue(data []byte) error {
	if c.isStopped() {
		return errors.New(fmt.Sprintf("connection %d is closed when send to queue", c.id))
	}
	c.initWriter()

	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	select {
	case c.msgChan <- data:
		return nil
	case <-idleTimeout.C:
		return errors.New(fmt.Sprintf("send buff msg %d is timeout", c.id))
	}
}

func (c *Connection) SendMsgBuff(t uint32, data []byte) error {
	if c.isStopped() {
		return errors.New(fmt.Sprintf("connection %d is closed when send msg buff", c.id))
	}

	b, err := c.getMsgData(t, data)

	if err != nil {
		return err
	}

	return c.SendToQueue(b)
}

// initWriter 初始化启动writer
func (c *Connection) initWriter() {
	if c.msgChan == nil && !c.writerSwitch.isStarted() && c.writerSwitch.setStarted() {
		c.msgChan = make(chan []byte, GlobalConfig.MaxMsgChanLen)

		go c.StartWriter()
	}
}

func (c *Connection) GetID() uint64 {
	return c.id
}

func (c *Connection) GetName() string {
	return c.name
}

func (c *Connection) SetUid(uid uint64) {
	c.uid = uid
}

func (c *Connection) GetUid() uint64 {
	return c.uid
}

func (c *Connection) GetConnection() net.Conn {
	return c.conn
}

func (c *Connection) GetTcpConnection() net.Conn {
	return c.conn
}

func (c *Connection) GetRemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Connection) GetLocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Connection) GetLocalAddrString() string {
	return c.conn.LocalAddr().String()
}

func (c *Connection) GetRemoteAddrString() string {
	return c.conn.RemoteAddr().String()
}

func (c *Connection) SetProperty(key string, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.property[key] = value
}

func (c *Connection) GetProperty(key string) (interface{}, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	v, ok := c.property[key]

	if !ok {
		return nil, errors.New(fmt.Sprintf("property key [%s] is undefined", key))
	}

	return v, nil
}

func (c *Connection) DeleteProperty(key string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, ok := c.property[key]
	if ok {
		delete(c.property, key)
	}
	return ok
}

func (c *Connection) GetWorker() IWorker {
	return c.worker
}

func (c *Connection) GetUseWorkerStatus() bool {
	return c.useWorkerStatus
}

func (c *Connection) GetWorkerId() uint32 {
	return c.workerId
}

func (c *Connection) SetWorkerId(id uint32) {
	c.workerId = id
}

// callOnConnStart 执行连接事件
func (c *Connection) callOnConnStart() {
	if c.onConnStart == nil {
		return
	}

	for _, fn := range c.onConnStart {
		fn(c)
	}
}

// callOnConnStop 执行断开连接事件
func (c *Connection) callOnConnStop() {
	if c.onConnStop == nil {
		return
	}

	for _, fn := range c.onConnStop {
		fn(c)
	}
}

func newConnection(server IServer, id uint64, conn net.Conn) *Connection {
	c := &Connection{
		id:              id,
		name:            server.ServerName(),
		conn:            conn,
		worker:          server.GetWorker(),
		useWorkerStatus: true,
	}

	frameDecode := server.GetFrameDecode()

	if frameDecode != nil {
		c.fd = frameDecode.New()
	}

	ctx, cancelFunc := context.WithCancel(server.Context())

	c.dp = server.GetDataPack()
	c.onConnStart = server.GetOnConnStart()
	c.onConnStop = server.GetOnConnStop()
	c.ctx = ctx
	c.cancel = cancelFunc

	return c
}

func newClientConnection(client IClient, conn net.Conn) *Connection {
	c := &Connection{
		name:   client.Name(),
		conn:   conn,
		worker: client.GetWorker(),
	}

	frameDecode := client.GetFrameDecode()

	if frameDecode != nil {
		c.fd = frameDecode.New()
	}

	ctx, cancelFunc := context.WithCancel(client.Context())

	c.dp = client.GetDataPack()
	c.onConnStart = client.GetOnConnStart()
	c.onConnStop = client.GetOnConnStop()
	c.ctx = ctx
	c.cancel = cancelFunc

	return c
}
