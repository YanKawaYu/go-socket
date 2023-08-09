package gosocket

import (
	"net"
	"sync"
	"time"
)

// Listener is a wrapper of net.TCPListener
type Listener struct {
	*net.TCPListener
	// record all the connections
	// 用于记录当前连接
	waitGroup *sync.WaitGroup
}

func NewListener(listener *net.TCPListener) *Listener {
	return &Listener{
		TCPListener: listener,
		waitGroup:   &sync.WaitGroup{},
	}
}

// WaitAllFinished Wait until all connections have closed
// 等待所有连接结束
func (listener *Listener) WaitAllFinished() {
	listener.waitGroup.Wait()
}

// Accept a new connection
func (listener *Listener) Accept() (net.Conn, error) {
	tcpConn, err := listener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	//Make sure the KeepAlive mechanism in TCP is opened
	//底层协议中也进行心跳保活
	tcpConn.SetKeepAlive(true)
	tcpConn.SetKeepAlivePeriod(time.Minute)
	//记录一个连接
	listener.waitGroup.Add(1)
	//Embed net.Conn in Connection to rewrite Close function
	//使用自定义的Connection嵌套net.Conn实例，以重写Close方法
	conn := &Connection{
		Conn:     tcpConn,
		listener: listener,
	}
	return conn, nil
}

// GetFd Get the fds to pass them to sub process
// 获取文件描述符，传递给子进程
func (listener *Listener) GetFd() (uintptr, error) {
	file, err := listener.File()
	if err != nil {
		return 0, err
	}
	return file.Fd(), nil
}
