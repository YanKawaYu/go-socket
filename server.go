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

// Server is the class to handle incoming requests by serving and listening
type Server struct {
	addr       string //监听地址 listen address
	isGraceful bool   //是否使用优雅重启 whether to use graceful restart

	signalChan chan os.Signal //接收重启信号的通道 the channel to receive restart signal
	listener   *Listener
}

// NewServer Create a new server
// 创建一个新的服务器
func NewServer(addr string, isGraceful bool) *Server {
	server := &Server{
		addr:       addr,
		isGraceful: isGraceful,
		signalChan: make(chan os.Signal),
	}
	return server
}

// ListenAndServe start listening and serving
// if the config isn't nil, then the tls will be enabled
// 启动服务器并开始监听，如果config不为nil，则启用TLS
func (server *Server) ListenAndServe(config *tls.Config) {
	listener := server.getTCPListener(TcpApp.Config.TcpPort)
	server.listener = NewListener(listener)

	restartManager := GetRestartManager()
	if restartManager != nil {
		//记录文件描述符 record the fds
		restartManager.MarkFd(kListenFd, listener)
		//监听重启 set a handler to listen to restart event
		restartManager.RegisterHandler(func() {
			//如果子进程启动成功，主进程停止接受连接
			//Stop main process from listening once the sub process has started
			server.listener.Close()
		})
	}
	//开始处理请求
	//Start handling requests
	server.serve(config)
}

// serve Start serving
// `config` pass nil to disable tls
func (server *Server) serve(config *tls.Config) {
	pid := os.Getpid()
	//Start to handle the connections
	for {
		acceptConn, err := server.listener.Accept()
		if config != nil {
			acceptConn = tls.Server(acceptConn, config)
		}
		if err != nil {
			//if the listener is closed, then it could be the child process has started
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				//stop the loop
				break
			}
			panic(err)
		}
		//For each connection, create a corresponding ClientConn instance to handle it
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
	close(server.signalChan)
}

// get tcp listener from a fd or a certain address
// 从文件描述符或者指定地址监听
func (server *Server) getTCPListener(port int) *net.TCPListener {
	var listener net.Listener
	var err error
	//If the server has restarted gracefully
	//Then the address and the port will be occupied, so the server should listen from the file that inherited from parent process
	//如果是优雅启动
	if server.isGraceful {
		//fd 3 is inherited from parent process, fd 0 is stdin, fd 1 is stdout, fd 2 is stderr
		//从父进程继承下来的文件描述符3监听，文件描述符012分别为stdin、stdout、stderr
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
