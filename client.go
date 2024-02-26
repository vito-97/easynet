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
	GetOnConnStart() []ConnCallback
	SetOnConnStop(callbacks ...ConnCallback)
	GetOnConnStop() []ConnCallback

	GetWorker() IWorker

	Context() context.Context

	GetDataPack() IDataPack
	GetDecode() IDecode
	GetFrameDecode() IFrameDecode
	SetDataPack(dp IDataPack)
	SetDecode(dc IDecode)
	SetFrameDecode(fd IFrameDecode)

	GetRouterManager() IRouterManager
}

type ClientOptionFunc = func(c *ClientOption)

type ClientOption struct {
	Name    string
	IP      string
	TCPPort int

	RouterManager IRouterManager

	Worker IWorker

	OnConnStart []ConnCallback
	OnConnStop  []ConnCallback

	Ctx    context.Context
	Cancel context.CancelFunc

	//接受到消息先处理的方法
	Handlers HandlersChain

	DataPack    IDataPack
	Decode      IDecode
	FrameDecode IFrameDecode
}

type Client struct {
	Switch
	restart Switch
	name    string

	conn IConnection

	IP      string
	TCPPort int

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
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.IP, c.TCPPort))

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

func (c *Client) GetOnConnStart() []ConnCallback {
	return c.onConnStart
}

func (c *Client) SetOnConnStop(callbacks ...ConnCallback) {
	if len(callbacks) > 0 {
		c.onConnStop = append(c.onConnStop, callbacks...)
	}
}

func (c *Client) GetOnConnStop() []ConnCallback {
	return c.onConnStop
}

func (c *Client) GetWorker() IWorker {
	return c.worker
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) GetDataPack() IDataPack {
	return c.dp
}

func (c *Client) GetDecode() IDecode {
	return c.dc
}

func (c *Client) GetFrameDecode() IFrameDecode {
	return c.fd
}

func (c *Client) SetDataPack(dp IDataPack) {
	c.dp = dp
}

func (c *Client) SetDecode(dc IDecode) {
	c.dc = dc
}

func (c *Client) SetFrameDecode(fd IFrameDecode) {
	c.fd = fd
}

func (c *Client) GetRouterManager() IRouterManager {
	return c
}

func DefaultClient(opts ...ClientOptionFunc) IClient {
	c := NewClient(opts...)

	c.Use(Logger(), Recovery())

	return c
}

func NewClient(optionFunc ...ClientOptionFunc) IClient {
	ctx, cancelFunc := context.WithCancel(context.Background())

	opts := &ClientOption{
		Name:    GlobalConfig.ClientName,
		IP:      GlobalConfig.Host,
		TCPPort: GlobalConfig.TCPPort,
		Ctx:     ctx,
		Cancel:  cancelFunc,
	}

	for _, fn := range optionFunc {
		fn(opts)
	}

	if opts.RouterManager == nil {
		opts.RouterManager = newRouterManager()
	}

	if opts.Worker == nil {
		opts.Worker = newWorker(opts.RouterManager, opts.Ctx, 0)
	}

	if opts.Decode == nil {
		opts.Decode = NewTLVDecoder()
	}

	if opts.FrameDecode == nil {
		opts.FrameDecode = NewFrameDecode(*opts.Decode.GetLengthField())
	}

	if opts.DataPack == nil {
		opts.DataPack = NewDataPack()
	}

	//读取数据中间件
	if opts.Decode != nil {
		opts.Handlers = append(opts.Handlers, opts.Decode.Handler())
	}

	c := &Client{
		name:           opts.Name,
		IP:             opts.IP,
		TCPPort:        opts.TCPPort,
		IRouterManager: opts.RouterManager,
		onConnStart:    opts.OnConnStart,
		onConnStop:     opts.OnConnStop,
		worker:         opts.Worker,
		ctx:            opts.Ctx,
		cancel:         opts.Cancel,
		dp:             opts.DataPack,
		dc:             opts.Decode,
		fd:             opts.FrameDecode,
	}

	//设置worker调度前调用的方法
	c.worker.WithHandler(opts.Handlers...)

	return c
}

func NewClientWithAddress(ip string, port int) IClient {
	return NewClient(ClientWithIP(ip), ClientWithPort(port))
}

func ClientWithIP(ip string) ClientOptionFunc {
	return func(c *ClientOption) {
		c.IP = ip
	}
}

func ClientWithPort(port int) ClientOptionFunc {
	return func(c *ClientOption) {
		c.TCPPort = port
	}
}

func ClientWithName(name string) ClientOptionFunc {
	return func(c *ClientOption) {
		c.Name = name
	}
}
