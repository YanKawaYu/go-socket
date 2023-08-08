package gosocket

import (
	"net"
	"sync"
)

// Connection is a wrapper for net.Conn
type Connection struct {
	net.Conn
	listener *Listener
	mutex    sync.Mutex

	// closed is used to mark whether the connection is closed
	// 连接是否已经关闭
	closed bool
}

func (conn *Connection) Close() error {
	// The lock is essential here, or else conn.listener.waitGroup.Done() might be called multiple times on concurrent call
	//必须用锁，否则并发的时候waitGroup的Done会被多次执行
	conn.mutex.Lock()
	if !conn.closed {
		conn.closed = true
		//Notify listener the connection is over
		//通知listener连接已结束
		conn.listener.waitGroup.Done()
	}
	conn.mutex.Unlock()
	return conn.Conn.Close()
}
