package gosocket

import (
	"net"
	"sync"
	"time"
)

type Listener struct {
	*net.TCPListener
	waitGroup *sync.WaitGroup //用于记录当前连接 record all the connections
}

func NewListener(listener *net.TCPListener) *Listener {
	return &Listener{
		TCPListener: listener,
		waitGroup:   &sync.WaitGroup{},
	}
}

// WaitAllFinished 等待所有连接结束
// Wait until all connections have closed
func (listener *Listener) WaitAllFinished() {
	listener.waitGroup.Wait()
}

func (listener *Listener) Accept() (net.Conn, error) {
	tcpConn, err := listener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	//底层协议中也进行心跳保活
	//Make sure the KeepAlive mechanism in TCP opened
	tcpConn.SetKeepAlive(true)
	tcpConn.SetKeepAlivePeriod(time.Minute)
	//记录一个连接
	listener.waitGroup.Add(1)
	//使用自定义的Connection嵌套net.Conn实例，以重写Close方法
	//Embed net.Conn in Connection to rewrite Close function
	conn := &Connection{
		Conn:     tcpConn,
		listener: listener,
	}
	return conn, nil
}

// GetFd 获取文件描述符，传递给子进程
// Get the fds to pass them to sub process
func (listener *Listener) GetFd() (uintptr, error) {
	file, err := listener.File()
	if err != nil {
		return 0, err
	}
	return file.Fd(), nil
}
