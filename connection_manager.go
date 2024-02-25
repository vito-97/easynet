package easynet

import (
	"errors"
	"fmt"
	"sync"
)

type IConnectionManager interface {
	Add(connection IConnection)
	Delete(connection IConnection)
	Get(uint64) (IConnection, error)
	GetByUid(uint64) ([]IConnection, error)
	DeleteByUid(uint64)
	Len() int
	Clear()
}

type ConnectionManager struct {
	server IServer

	list map[uint64]IConnection

	lock sync.RWMutex
}

func (c *ConnectionManager) Add(connection IConnection) {
	c.lock.Lock()
	defer c.lock.Unlock()
	id := connection.GetID()
	c.list[id] = connection
}

func (c *ConnectionManager) Delete(connection IConnection) {
	c.lock.Lock()
	defer c.lock.Unlock()
	id := connection.GetID()
	if _, ok := c.list[id]; !ok {
		return
	}
	delete(c.list, id)
}

func (c *ConnectionManager) Get(i uint64) (IConnection, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if conn, ok := c.list[i]; ok {
		return conn, nil
	}

	return nil, errors.New(fmt.Sprintf("connetion id %d not exists!", i))
}

func (c *ConnectionManager) GetByUid(u uint64) ([]IConnection, error) {
	var collect []IConnection
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, conn := range c.list {
		if conn.GetUid() == u {
			collect = append(collect, conn)
		}
	}

	if len(collect) == 0 {
		return nil, errors.New(fmt.Sprintf("connection uid %d not exists!", u))
	}

	return collect, nil
}

func (c *ConnectionManager) DeleteByUid(u uint64) {
	collect, err := c.GetByUid(u)

	if err != nil {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	for _, conn := range collect {
		delete(c.list, conn.GetID())
	}
}

func (c *ConnectionManager) Len() int {
	return len(c.list)
}

func (c *ConnectionManager) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.list = make(map[uint64]IConnection)
}

// newConnectionManager 获取连接管理器
func newConnectionManager(s IServer) *ConnectionManager {
	connMgr := &ConnectionManager{
		server: s,
		list:   make(map[uint64]IConnection),
	}

	return connMgr
}
