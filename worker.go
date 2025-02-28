package easynet

import (
	"context"
	"reflect"
	"sync"
)

// WorkerMode worker模式
type WorkerMode string

const (
	WorkerModeHash WorkerMode = "hash" //默认使用取余的方式
	WorkerModeBind WorkerMode = "bind" //为每个连接分配一个worker
)

type IWorker interface {
	Start()
	Stop()
	WithHandler(handlers ...HandlerFunc)
	// Use 使用worker
	Use(conn IConnection)
	// Free 归还worker
	Free(conn IConnection)
	// Execute 调度
	Execute(req IRequest)
}

type Worker struct {
	Switch
	size     uint32
	mode     WorkerMode
	taskLen  uint32
	handlers HandlersChain

	routerMgr IRouterManager

	//空闲的队列
	free     map[uint32]struct{}
	freeLock sync.Mutex
	//队列集合
	queue []chan IRequest

	ctx    context.Context
	cancel context.CancelFunc
}

func (w *Worker) WithHandler(handlers ...HandlerFunc) {
	if len(handlers) > 0 {
		w.handlers = append(w.handlers, handlers...)
	}
}

func (w *Worker) Start() {
	if !w.isStarted() && !w.setStarted() {
		return
	}

	debugPrint("worker is start\n")

	ctx, cancelFunc := context.WithCancel(context.Background())

	w.ctx = ctx
	w.cancel = cancelFunc

	w.queue = make([]chan IRequest, w.size)

	var free map[uint32]struct{}

	isBindMode := w.isBindMode()

	if isBindMode {
		free = make(map[uint32]struct{}, w.size)
	}

	for i := uint32(0); i < w.size; i++ {
		if isBindMode {
			free[i] = struct{}{}
		}

		w.queue[i] = make(chan IRequest, w.taskLen)

		go w.listenQueue(i)
	}

	w.free = free

	if !w.bindRouterHandlerExists() {
		w.handlers = append(w.handlers, BindRouterHandler(w.routerMgr))
	}
}

// bindRouterHandlerExists 绑定路由的方法是否存在
func (w *Worker) bindRouterHandlerExists() bool {
	for _, fn := range w.handlers {
		if reflect.ValueOf(fn).Pointer() == reflect.ValueOf(BindRouterHandler(w.routerMgr)).Pointer() {
			return true
		}
	}

	return false
}

func (w *Worker) Stop() {
	if !w.isStopped() && !w.setStopped() {
		return
	}

	for _, q := range w.queue {
		close(q)
	}

	w.queue = nil

	w.cancel()
}

func (w *Worker) listenQueue(i uint32) {
	q := w.queue[i]
	for !w.isStopped() {
		request, ok := <-q
		if !ok {
			debugPrint("worker id %d chan is closed\n", i)
			return
		}

		w.dispatch(request)
	}
}

// sendQueue 发送到chan队列中
func (w *Worker) sendQueue(request IRequest) {
	workerId := request.Connection().GetWorkerId()
	w.queue[workerId] <- request
}

func (w *Worker) Use(conn IConnection) {
	//不需要使用worker
	if !conn.GetUseWorkerStatus() {
		return
	}

	if w.isBindMode() {
		w.freeLock.Lock()
		defer w.freeLock.Unlock()

		for id := range w.free {
			delete(w.free, id)
			conn.SetWorkerId(id)
			return
		}
	}

	var workerId uint32

	if w.size == 0 {
		workerId = 0
	} else {
		workerId = uint32(conn.ID() % uint64(w.size))
	}

	conn.SetWorkerId(workerId)
}

func (w *Worker) Free(conn IConnection) {
	//不需要使用worker
	if !conn.GetUseWorkerStatus() {
		return
	}

	if !w.isBindMode() {
		return
	}

	w.freeLock.Lock()
	defer w.freeLock.Unlock()

	id := conn.GetWorkerId()

	w.free[id] = struct{}{}
}

func (w *Worker) Execute(req IRequest) {
	// 需要队列处理
	if w.size > 0 && req.Connection().GetUseWorkerStatus() {
		w.sendQueue(req)
	} else {
		go w.dispatch(req)
	}
}

// dispatch 调用请求函数
func (w *Worker) dispatch(req IRequest) {
	defer func() {
		if err := recover(); err != nil {
			debugPrint("worker dispatch error conn id = %d, type = %d, data = %s, error = %s\n", req.Connection().ID(), req.MsgType(), req.Data(), err)
		}
	}()

	for _, h := range w.handlers {
		h(req)
	}

	req.Next()
}

// BindRouterHandler 绑定路由中间件
func BindRouterHandler(routerMgr IRouterManager) HandlerFunc {
	return func(req IRequest) {
		req.SetHandler(routerMgr.GetHandlers(req.MsgType()))
	}
}

func (w *Worker) isBindMode() bool {
	return w.mode == WorkerModeBind
}

func (w *Worker) isHashMode() bool {
	return w.mode == "" || w.mode == WorkerModeHash
}

func newWorker(routerMgr IRouterManager, parentCtx context.Context, size ...uint32) *Worker {
	ctx, cancelFunc := context.WithCancel(parentCtx)

	w := &Worker{
		routerMgr: routerMgr,
		size:      GlobalConfig.WorkerPoolSize,
		taskLen:   GlobalConfig.MaxWorkerTaskLen,
		mode:      GlobalConfig.WorkerMode,
		ctx:       ctx,
		cancel:    cancelFunc,
	}

	if w.isBindMode() {
		w.size = uint32(GlobalConfig.MaxConn)
	}

	if len(size) > 0 {
		w.size = size[0]
	}

	return w
}
