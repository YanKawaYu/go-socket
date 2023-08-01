package gosocket

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	kListenFd = 3
)

type Server struct {
	addr       string //监听地址 listen address
	isGraceful bool   //是否使用优雅重启 whether to use graceful restart

	signalChan chan os.Signal //接收重启信号的通道 the channel to receive restart signal
	listener   *Listener
}

// NewServer 创建一个新的服务器
// Create a new server
func NewServer(addr string, isGraceful bool) *Server {
	server := &Server{
		addr:       addr,
		isGraceful: isGraceful,
		signalChan: make(chan os.Signal),
	}
	return server
}

// ListenAndServe 启动服务器并开始监听，如果config不为nil，则启用TLS
// start listening and serving
func (server *Server) ListenAndServe(config *tls.Config) {
	listener := server.getTCPListener(TcpApp.Config.TcpPort)
	server.listener = NewListener(listener)
	//记录文件描述符 record the fds
	GetRestartManager().MarkFd(kListenFd, listener)
	//监听重启 set a handler to listen to restart event
	GetRestartManager().RegisterHandler(func() {
		//如果子进程启动成功，主进程停止接受连接
		//Stop main process from listening once the sub process has started
		server.listener.Close()
	})
	//开始处理请求
	//Start handling requests
	server.serve(config)
}

// 如果config不为nil，则启用TLS
func (server *Server) serve(config *tls.Config) {
	pid := os.Getpid()
	//开始处理连接
	for {
		acceptConn, err := server.listener.Accept()
		if config != nil {
			acceptConn = tls.Server(acceptConn, config)
		}
		if err != nil {
			//如果listener被关闭，说明子进程已经启动，直接跳出循环即可
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				break
			}
			panic(err)
		}
		client := NewClientConn(acceptConn)
		if client != nil {
			//开始读和写队列
			//Start the read and write queue
			client.Start()
		} else {
			//初始化连接失败，直接关闭
			//if init conn failed
			acceptConn.Close()
		}
	}
	//等待所有连接都结束后再结束进程
	//Wait until all connections have closed
	server.listener.WaitAllFinished()
	fmt.Printf("All connection were closed, process %d is shutting down...\n", pid)
	//关闭信号通道
	close(server.signalChan)
}

// 从文件描述符或者指定地址监听
func (server *Server) getTCPListener(port int) *net.TCPListener {
	var listener net.Listener
	var err error
	//如果是优雅启动 if graceful restart
	if server.isGraceful {
		//从父进程继承下来的文件描述符3监听，文件描述符012分别为stdin、stdout、stderr
		//3 fds that are inherited from parent process, including stdin, stdout, stderr
		file := os.NewFile(kListenFd, "")
		listener, err = net.FileListener(file)
		if err != nil {
			panic(err)
		}
	} else {
		address := server.addr
		if port > 0 {
			address += ":" + strconv.Itoa(port)
		}
		//从指定端口监听
		listener, err = net.Listen("tcp", address)
		if err != nil {
			panic(err)
		}
	}
	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		panic("get listener failed")
	}
	return tcpListener
}
