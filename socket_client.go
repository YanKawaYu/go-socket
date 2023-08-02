package gosocket

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	ip     string
	port   int
	isTls  bool
	logger ILogger

	conn *SocketClientConn

	pingTimer *Timer
}

// NewClient create a new client by providing the ip, port of the server and whether to use tls
// 创建一个新的客户端连接
func NewClient(ip string, port int, isTls bool, log ILogger) *Client {
	c := &Client{
		ip:     ip,
		port:   port,
		isTls:  isTls,
		logger: log,
	}
	return c
}

// GetConnectInfo Override this function to provide connect info string to the server
// 重载这个函数向服务器提供连接信息
func (client *Client) GetConnectInfo() string {
	return ""
}

// Connect start to connect the server
// 连接聊天服务器
func (client *Client) Connect() (err error) {
	defer func() {
		if recoverObj := recover(); recoverObj != nil {
			client.logger.Error(recoverObj)
		}
	}()
	if client.port <= 0 {
		panic("port needs to be above 0")
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

// GetDataCallback is the callback used by GetData function
type GetDataCallback func(err error, data string)

// GetData Call apis of the server
// 调用服务器的接口
//
// payloadType should contain two parts, for example 'chat.AddMessage'
// The first part corresponds to controller name
// The second part corresponds to action name
// In this way, the request will be routed to the certain action under certain controller automatically
//
// Payload should be the main content that is sent to the server, which will be encoded into json
func (client *Client) GetData(payloadType string, payload interface{}, callback GetDataCallback, data []byte) {
	payloadStr := ""
	if payload != nil {
		payloadStr = JSONEncode(payload)
	}
	if client.conn == nil {
		if callback != nil {
			callback(errors.New("connect required"), "")
		}
		return
	}
	//加锁，确保计时器结束和接口返回不会出现并发
	timeOutLock := &sync.RWMutex{}
	isCallback := false
	//启动计时器，如果一段时间没有收到服务器响应，则返回超时错误
	timer := NewTimer(time.Second*10, func() {
		timeOutLock.Lock()
		defer timeOutLock.Unlock()
		//如果已经收到服务器响应了，直接返回
		if isCallback {
			return
		} else {
			isCallback = true
		}
		if callback != nil {
			callback(errors.New("timeout"), "")
		}
	})
	client.conn.SendRequest(payloadType, payloadStr, func(payloadBody string) {
		timeOutLock.Lock()
		defer timeOutLock.Unlock()
		defer func() {
			if r := recover(); r != nil {
				client.logger.Error(r)
			}
		}()
		//如果已超时，直接返回
		if isCallback {
			return
		} else {
			isCallback = true
			//停止计时器
			timer.Stop()
		}
		err, ret := client.DecodeResponse(payloadBody)
		if callback != nil {
			callback(err, ret)
		}
	}, data)
}

type ClientResponseBody struct {
	Status  Status           `json:"status"`
	Message string           `json:"message,omitempty"`
	Data    *json.RawMessage `json:"data,omitempty"`
}

// DecodeResponse Override this function to decode server response
// 重载这个方法对服务器的返回进行解析
func (client *Client) DecodeResponse(payloadBody string) (error, string) {
	result := ""
	respBody := ClientResponseBody{}
	JSONDecode(payloadBody, &respBody)
	if respBody.Status == StatusSuccess {
		if respBody.Data != nil {
			dataBytes, err := respBody.Data.MarshalJSON()
			if err == nil {
				result = string(dataBytes)
			} else {
				return errors.New("response data error"), ""
			}
		}
	} else {
		return errors.New("response status error"), ""
	}
	return nil, result
}

// Disconnect from server
// 断开与服务器的连接
func (client *Client) Disconnect() {
	//停止心跳包
	client.stopAutoPing()
	//断开连接
	client.conn.Disconnect()
	client.conn = nil
}

// Start ping pong
// 开始心跳
func (client *Client) startAutoPing() {
	client.pingTimer = NewTimer(60*time.Second, func() {
		if client.conn != nil {
			client.conn.SendPing()
		}
	})
}

// Stop ping pong
// 结束心跳
func (client *Client) stopAutoPing() {
	//防止重复调用
	if client.pingTimer != nil {
		client.pingTimer.Stop()
		client.pingTimer = nil
	}
}

// OnSendReqReceived Override this function to handle the push notification from server
// 收到服务器推送
// ClientConnInterface
func (client *Client) OnSendReqReceived(reqType string, reqBody string) {}

// OnDisconnect Handle issues after the connection is off
// 连接已断开
// ClientConnInterface
func (client *Client) OnDisconnect() {
	client.stopAutoPing()
}
