package easynet

import (
	"context"
	"fmt"
	"net"
)

type IClient interface {
	IRouterManager

	Run()
	Restart()
	Stop()
	Name() string

	Conn() IConnection

	SetOnConnStart(callbacks ...ConnCallback)
	OnConnStart() []ConnCallback
	SetOnConnStop(callbacks ...ConnCallback)
	OnConnStop() []ConnCallback

	Worker() IWorker

	Context() context.Context

	DataPack() IDataPack
	Decode() IDecode
	FrameDecode() IFrameDecode

	RouterManager() IRouterManager
}

type ClientOption = func(c *Client)

type Client struct {
	Switch
	restart Switch
	name    string

	conn IConnection

	ip   string
	port int

	// 路由管理
	IRouterManager

	onConnStart []ConnCallback
	onConnStop  []ConnCallback

	ctx    context.Context
	cancel context.CancelFunc

	worker IWorker

	dp IDataPack
	dc IDecode
	fd IFrameDecode
}

func (c *Client) Run() {
	if c.isStarted() || !c.setStarted() {
		debugPrint("client name [%s] is running!\n", c.name)
		return
	}

	if !c.restart.isStarted() {
		c.worker.Start()
	}

	go c.start()
}

func (c *Client) Restart() {
	if c.restart.isStarted() || !c.restart.setStarted() {
		debugPrint("client name [%s] is restarting!\n", c.name)
		return
	}
	c.Stop()
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.ctx = ctx
	c.cancel = cancelFunc
	c.reset()
	c.Run()
	c.restart.reset()
}

func (c *Client) start() {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.ip, c.port))

	if err != nil {
		panic(fmt.Sprintf("client name [%s] listen error:%v", c.name, err))
	}

	//初始化客户端
	connection := newClientConnection(c, conn)
	//启动客户端
	go c.startConn(connection)

	select {
	case <-c.ctx.Done():
		debugPrint("client name [%s] is stopped!\n", c.name)
		err := conn.Close()
		if err != nil {
			debugPrint("client close error:%v\n", err)
		}
	}
}

func (c *Client) startConn(conn IConnection) {
	c.conn = conn
	conn.Start()
}

func (c *Client) Stop() {
	if c.isStopped() {
		return
	}
	if !c.setStopped() {
		return
	}
	if !c.restart.isStarted() {
		c.worker.Stop()
	}
	c.cancel()
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Conn() IConnection {
	return c.conn
}

func (c *Client) SetOnConnStart(callbacks ...ConnCallback) {
	if len(callbacks) > 0 {
		c.onConnStart = append(c.onConnStart, callbacks...)
	}
}

func (c *Client) OnConnStart() []ConnCallback {
	return c.onConnStart
}

func (c *Client) SetOnConnStop(callbacks ...ConnCallback) {
	if len(callbacks) > 0 {
		c.onConnStop = append(c.onConnStop, callbacks...)
	}
}

func (c *Client) OnConnStop() []ConnCallback {
	return c.onConnStop
}

func (c *Client) Worker() IWorker {
	return c.worker
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) DataPack() IDataPack {
	return c.dp
}

func (c *Client) Decode() IDecode {
	return c.dc
}

func (c *Client) FrameDecode() IFrameDecode {
	return c.fd
}
func (c *Client) RouterManager() IRouterManager {
	return c
}

func DefaultClient(opts ...ClientOption) IClient {
	c := NewClient(opts...)

	c.Use(Logger(), Recovery())

	return c
}

func NewClient(optionFunc ...ClientOption) IClient {
	ctx, cancelFunc := context.WithCancel(context.Background())

	c := &Client{
		name:   GlobalConfig.ClientName,
		ip:     GlobalConfig.Host,
		port:   GlobalConfig.Port,
		ctx:    ctx,
		cancel: cancelFunc,
	}

	for _, fn := range optionFunc {
		fn(c)
	}

	if c.IRouterManager == nil {
		c.IRouterManager = newRouterManager()
	}

	if c.worker == nil {
		c.worker = newWorker(c.ctx, c.IRouterManager, 0)
	}

	if c.dc == nil {
		c.dc = NewTLVDecoder()
	}

	if c.fd == nil {
		c.fd = NewFrameDecode(*c.dc.LengthField())
	}

	if c.dp == nil {
		c.dp = NewDataPack()
	}

	//设置worker调度前调用的方法
	c.worker.WithHandler(c.dc.Handler())

	return c
}

func NewClientWithAddress(ip string, port int) IClient {
	return NewClient(ClientWithIP(ip), ClientWithPort(port))
}

func ClientWithIP(ip string) ClientOption {
	return func(c *Client) {
		c.ip = ip
	}
}

func ClientWithPort(port int) ClientOption {
	return func(c *Client) {
		c.port = port
	}
}

func ClientWithName(name string) ClientOption {
	return func(c *Client) {
		c.name = name
	}
}

func ClientWithDataPack(dp IDataPack) ClientOption {
	return func(c *Client) {
		c.dp = dp
	}
}

func ClientWithDecode(dc IDecode) ClientOption {
	return func(c *Client) {
		c.dc = dc
	}
}

func ClientWithFrameDecode(fd IFrameDecode) ClientOption {
	return func(c *Client) {
		c.fd = fd
	}
}

func ClientWithWorker(worker IWorker) ClientOption {
	return func(c *Client) {
		c.worker = worker
	}
}

func ClientWithOnConnStart(callbacks ...ConnCallback) ClientOption {
	return func(c *Client) {
		c.onConnStart = append(c.onConnStart, callbacks...)
	}
}

func ClientWithOnConnStop(callbacks ...ConnCallback) ClientOption {
	return func(c *Client) {
		c.onConnStop = append(c.onConnStop, callbacks...)
	}
}

func ClientWithRouterManager(routerMgr IRouterManager) ClientOption {
	return func(c *Client) {
		c.IRouterManager = routerMgr
	}
}

func ClientWithContext(ctx context.Context) ClientOption {
	return func(c *Client) {
		c.ctx = ctx
	}
}

func ClientWithCancel(cancel context.CancelFunc) ClientOption {
	return func(c *Client) {
		c.cancel = cancel
	}
}
