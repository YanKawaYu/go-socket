package gotcp

import (
	"net"
	"sync"
)

type Connection struct {
	net.Conn
	listener *Listener
	mutex sync.Mutex

	closed bool //连接是否已经关闭
}

func (conn *Connection) Close() error {
	//必须用锁，否则并发的时候waitGroup的Done会被多次执行
	conn.mutex.Lock()
	if !conn.closed {
		conn.closed = true
		//通知listener连接已结束
		conn.listener.waitGroup.Done()
	}
	conn.mutex.Unlock()
	return conn.Conn.Close()
}