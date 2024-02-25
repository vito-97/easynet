package easynet

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

type ConnCallback func(connection IConnection)

type IServer interface {
	IRouterManager

	Stop()
	Run()
	GetWorker() IWorker
	ServerName() string

	GetConnectionManager() IConnectionManager

	SetOnConnStart(callbacks ...ConnCallback)
	GetOnConnStart() []ConnCallback
	SetOnConnStop(callbacks ...ConnCallback)
	GetOnConnStop() []ConnCallback

	Context() context.Context

	GetDataPack() IDataPack
	GetDecode() IDecode
	GetFrameDecode() IFrameDecode
	SetDataPack(dp IDataPack)
	SetDecode(dc IDecode)
	SetFrameDecode(fd IFrameDecode)

	GetRouterManager() IRouterManager
}

type ServerOptionFunc = func(s *ServerOption)

type ServerOption struct {
	Name string

	IP      string
	TCPPort int

	// 路由管理
	RouterManager IRouterManager
	ConnMgr       IConnectionManager

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

type Server struct {
	Switch
	name string

	IP      string
	TCPPort int

	// 路由管理
	IRouterManager
	connMgr IConnectionManager

	worker IWorker

	onConnStart []ConnCallback
	onConnStop  []ConnCallback

	ctx    context.Context
	cancel context.CancelFunc

	clientID uint64

	dp IDataPack
	dc IDecode
	fd IFrameDecode
}

func (s *Server) Run() {
	if s.isStarted() || !s.setStarted() {
		debugPrint("server name [%s] is started!\n", s.name)
		return
	}

	go s.start()

	s.worker.Start()

	var sc = make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-s.ctx.Done():
		debugPrint("server name [%s] is stopped!\n", s.name)
	case sig := <-sc:
		debugPrint("server name [%s], serve interrupt, signal = %v\n", s.name, sig)
		s.Stop()
	}
}

func (s *Server) start() {
	var delay Delay
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.IP, s.TCPPort))

	if err != nil {
		panic(fmt.Sprintf("server name [%s] listen error:%v", s.name, err))
	}

	go func() {
		for {
			if s.connMgr.Len() >= GlobalConfig.MaxConn {
				delay.Sleep()
				continue
			}

			conn, err := listen.Accept()

			if err != nil {
				debugPrint("accept error:%v\n", err)
				return
			}
			delay.Reset()
			//获取客户端ID
			cid := atomic.AddUint64(&s.clientID, 1)
			//初始化客户端
			connection := newConnection(s, cid, conn)
			//启动客户端
			go s.startConn(connection)
		}
	}()

	select {
	case <-s.ctx.Done():
		err := listen.Close()
		if err != nil {
			debugPrint("listener close error:%v\n", err)
		}
	}
}

func (s *Server) startConn(conn IConnection) {
	s.worker.Use(conn)
	s.connMgr.Add(conn)
	conn.Start()
}

func (s *Server) stopConn(conn IConnection) {
	//释放绑定的处理队列
	s.worker.Free(conn)
	//连接管理移除该连接
	s.connMgr.Delete(conn)
}

func (s *Server) Stop() {
	if s.isStopped() {
		return
	}
	if !s.setStopped() {
		return
	}
	s.connMgr.Clear()
	s.worker.Stop()
	s.cancel()
}

func (s *Server) ServerName() string {
	return s.name
}

func (s *Server) GetConnectionManager() IConnectionManager {
	return s.connMgr
}

func (s *Server) SetOnConnStart(callbacks ...ConnCallback) {
	if len(callbacks) > 0 {
		s.onConnStart = append(s.onConnStart, callbacks...)
	}
}

func (s *Server) GetOnConnStart() []ConnCallback {
	return s.onConnStart
}

func (s *Server) SetOnConnStop(callbacks ...ConnCallback) {
	if len(callbacks) > 0 {
		s.onConnStop = append(s.onConnStop, callbacks...)
	}
}

func (s *Server) GetOnConnStop() []ConnCallback {
	return s.onConnStop
}

func (s *Server) GetDataPack() IDataPack {
	return s.dp
}

func (s *Server) GetDecode() IDecode {
	return s.dc
}

func (s *Server) GetFrameDecode() IFrameDecode {
	return s.fd
}

func (s *Server) SetDataPack(dp IDataPack) {
	s.dp = dp
}

func (s *Server) SetDecode(dc IDecode) {
	s.dc = dc
}

func (s *Server) SetFrameDecode(fd IFrameDecode) {
	s.fd = fd
}

func (s *Server) Context() context.Context {
	return s.ctx
}

func (s *Server) GetWorker() IWorker {
	return s.worker
}

func (s *Server) GetRouterManager() IRouterManager {
	return s
}

func New(optionFunc ...ServerOptionFunc) *Server {
	ctx, cancelFunc := context.WithCancel(context.Background())

	opts := &ServerOption{
		Name:    GlobalConfig.Name,
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
		opts.Worker = newWorker(opts.RouterManager, opts.Ctx)
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

	s := &Server{
		name:           opts.Name,
		IP:             opts.IP,
		TCPPort:        opts.TCPPort,
		worker:         opts.Worker,
		IRouterManager: opts.RouterManager,
		onConnStart:    opts.OnConnStart,
		onConnStop:     opts.OnConnStop,
		ctx:            opts.Ctx,
		cancel:         opts.Cancel,
		dp:             opts.DataPack,
		dc:             opts.Decode,
		fd:             opts.FrameDecode,
	}

	if opts.ConnMgr == nil {
		s.connMgr = newConnectionManager(s)
	} else {
		s.connMgr = opts.ConnMgr
	}

	// 断开连接时把保存在连接管理中的连接移除
	s.SetOnConnStop(s.stopConn)

	//设置worker调度前调用的方法
	s.worker.WithHandler(opts.Handlers...)

	return s
}

func Default(opts ...ServerOptionFunc) *Server {
	s := New(opts...)

	s.Use(Logger(), Recovery())

	return s
}

func ServerWithIP(ip string) ServerOptionFunc {
	return func(c *ServerOption) {
		c.IP = ip
	}
}

func ServerWithPort(port int) ServerOptionFunc {
	return func(c *ServerOption) {
		c.TCPPort = port
	}
}

func ServerWithName(name string) ServerOptionFunc {
	return func(c *ServerOption) {
		c.Name = name
	}
}
