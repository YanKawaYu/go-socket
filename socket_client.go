package gosocket

import (
	"crypto/tls"
	"net"
	"strconv"
	"time"
)

type Client struct {
	ip     string
	port   int
	isTls  bool
	logger ILogger

	conn *SocketClientConn

	Uid       int64
	pingTimer *Timer
}

func NewClient(ip string, port int, isTls bool, log ILogger) *Client {
	c := &Client{
		ip:     ip,
		port:   port,
		isTls:  isTls,
		logger: log,
	}
	return c
}

func (client *Client) GetConnectInfo() string {
	return ""
}

// 连接聊天服务器
func (client *Client) Connect() (err error) {
	defer func() {
		if recoverObj := recover(); recoverObj != nil {
			client.logger.Error(recoverObj)
		}
	}()
	if client.port <= 0 {
		panic("port必须大于0哦")
	}
	//连接服务器
	var connection net.Conn
	addr := client.ip + ":" + strconv.Itoa(client.port)
	if client.isTls {
		config := &tls.Config{
			InsecureSkipVerify: true,
		}
		connection, err = tls.Dial("tcp", addr, config)
	} else {
		connection, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return
	}
	client.conn = NewSocketClientConn(connection, client.logger)
	err = client.conn.Connect(client.GetConnectInfo())
	//如果连接成功
	if err == nil {
		//每隔一段时间发送心跳包
		client.startAutoPing()
	}
	return
}

func (client *Client) Disconnect() {
	//停止心跳包
	client.stopAutoPing()
	//断开连接
	client.conn.Disconnect()
	client.conn = nil
}

// 开始心跳
func (client *Client) startAutoPing() {
	client.pingTimer = NewTimer(60*time.Second, func() {
		if client.conn != nil {
			client.conn.SendPing()
		}
	})
}

// 结束心跳
func (client *Client) stopAutoPing() {
	//防止重复调用
	if client.pingTimer != nil {
		client.pingTimer.Stop()
		client.pingTimer = nil
	}
}
