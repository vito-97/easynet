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
	Worker() IWorker
	Name() string

	ConnectionManager() IConnectionManager

	SetOnConnStart(callbacks ...ConnCallback)
	OnConnStart() []ConnCallback
	SetOnConnStop(callbacks ...ConnCallback)
	OnConnStop() []ConnCallback

	Context() context.Context

	DataPack() IDataPack
	Decode() IDecode
	FrameDecode() IFrameDecode

	RouterManager() IRouterManager
}

type ServerOption = func(s *Server)

type Server struct {
	Switch
	name string

	ip   string
	port int

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
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.ip, s.port))

	if err != nil {
		panic(fmt.Sprintf("server name [%s] listen error:%v", s.name, err))
	}

	go func() {
		for !s.isStopped() {
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

func (s *Server) Name() string {
	return s.name
}

func (s *Server) ConnectionManager() IConnectionManager {
	return s.connMgr
}

func (s *Server) SetOnConnStart(callbacks ...ConnCallback) {
	if len(callbacks) > 0 {
		s.onConnStart = append(s.onConnStart, callbacks...)
	}
}

func (s *Server) OnConnStart() []ConnCallback {
	return s.onConnStart
}

func (s *Server) SetOnConnStop(callbacks ...ConnCallback) {
	if len(callbacks) > 0 {
		s.onConnStop = append(s.onConnStop, callbacks...)
	}
}

func (s *Server) OnConnStop() []ConnCallback {
	return s.onConnStop
}

func (s *Server) DataPack() IDataPack {
	return s.dp
}

func (s *Server) Decode() IDecode {
	return s.dc
}

func (s *Server) FrameDecode() IFrameDecode {
	return s.fd
}

func (s *Server) Context() context.Context {
	return s.ctx
}

func (s *Server) Worker() IWorker {
	return s.worker
}

func (s *Server) RouterManager() IRouterManager {
	return s
}

func New(optionFunc ...ServerOption) *Server {
	ctx, cancelFunc := context.WithCancel(context.Background())

	s := &Server{
		name:   GlobalConfig.Name,
		ip:     GlobalConfig.Host,
		port:   GlobalConfig.Port,
		ctx:    ctx,
		cancel: cancelFunc,
	}

	for _, fn := range optionFunc {
		fn(s)
	}

	if s.IRouterManager == nil {
		s.IRouterManager = newRouterManager()
	}

	if s.worker == nil {
		s.worker = newWorker(s.ctx, s.IRouterManager)
	}

	if s.dc == nil {
		s.dc = NewTLVDecoder()
	}

	if s.fd == nil {
		s.fd = NewFrameDecode(*s.dc.LengthField())
	}

	if s.dp == nil {
		s.dp = NewDataPack()
	}

	if s.connMgr == nil {
		s.connMgr = newConnectionManager(s)
	}

	// 断开连接时把保存在连接管理中的连接移除
	s.SetOnConnStop(s.stopConn)

	//设置worker调度前调用的方法
	s.worker.WithHandler(s.dc.Handler())

	return s
}

func DefaultServer(opts ...ServerOption) *Server {
	s := New(opts...)

	s.Use(Logger(), Recovery())

	return s
}

func ServerWithIP(ip string) ServerOption {
	return func(c *Server) {
		c.ip = ip
	}
}

func ServerWithPort(port int) ServerOption {
	return func(c *Server) {
		c.port = port
	}
}

func ServerWithName(name string) ServerOption {
	return func(c *Server) {
		c.name = name
	}
}

func ServerWithRouterManager(routerMgr IRouterManager) ServerOption {
	return func(c *Server) {
		c.IRouterManager = routerMgr
	}
}

func ServerWithWorker(worker IWorker) ServerOption {
	return func(c *Server) {
		c.worker = worker
	}
}

func ServerWithConnMgr(connMgr IConnectionManager) ServerOption {
	return func(c *Server) {
		c.connMgr = connMgr
	}
}

func ServerWithDecode(dc IDecode) ServerOption {
	return func(c *Server) {
		c.dc = dc
	}
}

func ServerWithFrameDecode(fd IFrameDecode) ServerOption {
	return func(c *Server) {
		c.fd = fd
	}
}

func ServerWithDataPack(dp IDataPack) ServerOption {
	return func(c *Server) {
		c.dp = dp
	}
}

func ServerWithOnConnStart(callbacks ...ConnCallback) ServerOption {
	return func(c *Server) {
		c.onConnStart = callbacks
	}
}

func ServerWithOnConnStop(callbacks ...ConnCallback) ServerOption {
	return func(c *Server) {
		c.onConnStop = callbacks
	}
}

func ServerWithContext(ctx context.Context) ServerOption {
	return func(c *Server) {
		c.ctx = ctx
	}
}

func ServerWithCancel(cancel context.CancelFunc) ServerOption {
	return func(c *Server) {
		c.cancel = cancel
	}
}
