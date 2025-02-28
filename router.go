package easynet

import (
	"fmt"
	"sync"
)

type IRouterGroup interface {
	// Add 添加该消息类型的处理方法
	Add(t uint32, handlers ...HandlerFunc)
	// Use 全局中间件
	Use(handlers ...HandlerFunc) IRouterGroup
	// Group 分组
	Group(name string, handlers ...HandlerFunc) IRouterGroup
	// Middleware 获取中间件
	Middleware() []HandlerFunc
	// Manager 获取路由管理
	Manager() IRouterManager
}

type IRouterManager interface {
	IRouterGroup
	GetHandlers(t uint32) []HandlerFunc
}

type RouterManager struct {
	*RouterGroup

	lock sync.RWMutex
	// 保存所有路由
	routers map[uint32]HandlersChain
	// 所有路由分组
	routerGroups map[string]*RouterGroup
}

var _ IRouterManager = (*RouterManager)(nil)
var _ IRouterGroup = (*RouterGroup)(nil)

func (r *RouterManager) Add(t uint32, handler ...HandlerFunc) {
	if _, ok := r.routers[t]; ok {
		panic(fmt.Sprintf("route id %v is exixts!", t))
		return
	}

	r.routers[t] = combineHandlers(r.middleware, handler)

	debugPrintRoute(t, r.routers[t])
}

func (r *RouterManager) Group(name string, handlers ...HandlerFunc) IRouterGroup {
	if rg, ok := r.routerGroups[name]; ok {
		return rg
	}

	rg := &RouterGroup{
		name:       name,
		mgr:        r,
		middleware: handlers,
	}

	r.routerGroups[name] = rg

	return rg
}

func (r *RouterManager) GetHandlers(t uint32) []HandlerFunc {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.routers[t]
}

type RouterGroup struct {
	name       string
	middleware HandlersChain
	mgr        *RouterManager
}

func (r *RouterGroup) Add(t uint32, handlers ...HandlerFunc) {
	r.mgr.Add(t, combineHandlers(r.middleware, handlers)...)
}

func (r *RouterGroup) Use(handlers ...HandlerFunc) IRouterGroup {
	r.middleware = append(r.middleware, handlers...)

	return r
}

func (r *RouterGroup) Group(name string, handlers ...HandlerFunc) IRouterGroup {
	rg := r.mgr.Group(name, combineHandlers(r.middleware, handlers)...)

	return rg
}

func (r *RouterGroup) Middleware() []HandlerFunc {
	return r.middleware
}

func (r *RouterGroup) Manager() IRouterManager {
	return r.mgr
}

func newRouterManager() *RouterManager {
	mgr := &RouterManager{
		routers:      make(map[uint32]HandlersChain, 10),
		routerGroups: make(map[string]*RouterGroup, 10),
	}

	mgr.RouterGroup = newRouterGroup("root", mgr)

	return mgr
}

func newRouterGroup(name string, mgr *RouterManager) *RouterGroup {
	return &RouterGroup{
		name: name,
		mgr:  mgr,
	}
}

// combineHandlers 合并两个切片
func combineHandlers(middlewares HandlersChain, handlers HandlersChain) HandlersChain {
	size := len(middlewares) + len(handlers)
	mergedHandlers := make(HandlersChain, size)

	copy(mergedHandlers, middlewares)
	copy(mergedHandlers[len(middlewares):], handlers)

	return mergedHandlers
}
