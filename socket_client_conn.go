package gosocket

import (
	"github.com/yankawayu/go-socket/packet"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

const QueueLength = 50

type SendReqCallback func(payloadBody string)

type ClientConnInterface interface {
	// OnSendReqReceived called once there is a push notification from the server
	OnSendReqReceived(reqType string, reqBody string)
	// OnDisconnect called once the connection is off
	OnDisconnect()
}

// SocketClientConn is a class inside Client responsible for connecting to the server
type SocketClientConn struct {
	cInterface ClientConnInterface
	conn       net.Conn
	jobChan    chan Job //任务队列

	// connAckChan is the queue for ConnAck message
	//连接回复队列
	connAckChan chan *packet.ConnAck

	// reqMsgId is an autoincrement message id
	//协议自增消息id
	reqMsgId  uint16
	msgIdLock *sync.RWMutex

	// reqMsgMap is used to store all the message callbacks
	//等待回复的消息map
	reqMsgMap map[uint16]SendReqCallback
	mapLock   *sync.RWMutex

	msgManager *packet.MessageManager //协议层的包管理器
	log        ILogger                //输出日志用
}

func NewSocketClientConn(connection net.Conn, log ILogger) *SocketClientConn {
	cli := &SocketClientConn{
		conn:        connection,
		jobChan:     make(chan Job, QueueLength),
		connAckChan: make(chan *packet.ConnAck),
		reqMsgId:    1,
		msgIdLock:   &sync.RWMutex{},
		reqMsgMap:   make(map[uint16]SendReqCallback),
		mapLock:     &sync.RWMutex{},

		msgManager: &packet.MessageManager{
			ProCommon: packet.ProtocolCommon{
				ProName:           packet.ProtocolName,    //协议名
				ProVersion:        packet.ProtocolVersion, //协议版本号
				KeepAliveTime:     60,                     //心跳包间隔
				EnablePayloadGzip: true,                   //是否开启gzip
			},
		},
		log: log,
	}
	go cli.startReader()
	go cli.startWriter()
	return cli
}

func (client *SocketClientConn) SetConnInterface(connInterface ClientConnInterface) {
	client.cInterface = connInterface
}

func (client *SocketClientConn) GetConnInterface() ClientConnInterface {
	return client.cInterface
}

func (client *SocketClientConn) startReader() {
	defer func() {
		close(client.jobChan)
		client.conn.Close()
		if client.cInterface != nil {
			client.cInterface.OnDisconnect()
		}
		//log.Println("reader stopped")
	}()
	for {
		//log.Println("start waiting to read")
		//获取消息
		msg, err := client.msgManager.DecodeMessage(client.conn)
		if err != nil {
			//Whether the server close the connection
			//捕获到服务器关闭连接的信号
			if err == io.EOF {
				log.Println("server close connection")
				return
			}
			//连接已被关闭
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}
			log.Println("get message fail:", err)
			return
		}
		//Handle the message differently according to its type
		//根据消息类型进行不同处理
		switch msg := msg.(type) {
		case *packet.ConnAck:
			client.connAckChan <- msg
		case *packet.PingResp:
			//log.Println("got ping response")
		case *packet.SendResp:
			client.handleSendResp(msg.MessageId, msg.Payload)
		case *packet.SendReq:
			//收到服务器推送的SyncKey变化
			client.handleSendReq(msg.Type, msg.Payload)
		case *packet.Disconnect:
			log.Println("receive disconnect")
			return
		default:
			log.Printf("unknown message type %T", msg)
		}
	}
}

func (client *SocketClientConn) startWriter() {
	defer func() {
		//log.Println("writer stopped")
	}()
	for job := range client.jobChan {
		//log.Println("got job")
		err := client.msgManager.EncodeMessage(client.conn, job.Message)
		//通知消息发送完成
		if job.Receipt != nil {
			close(job.Receipt)
		}
		if err != nil {
			log.Println("write error", err)
			return
		}
		//确保发完Disconnect消息马上结束
		if _, ok := job.Message.(*packet.Disconnect); ok {
			return
		}
		//log.Println("finish writing")
	}
}

func (client *SocketClientConn) Connect(loginInfo string) error {
	connectMsg := &packet.Connect{
		Payload: loginInfo,
	}
	//Add the message into the queue and block until the message is sent
	//将消息加入任务队列，阻塞直到消息发送完成
	client.sync(connectMsg)
	//Block again until there is a ConnAck message
	//This is how the connect message works
	//阻塞等待连接回复
	ack := <-client.connAckChan
	return packet.ConnectionErrors[ack.ReturnCode]
}

func (client *SocketClientConn) Disconnect() {
	disconnectMsg := &packet.Disconnect{}
	client.submit(disconnectMsg)
}

func (client *SocketClientConn) SendRequest(payloadType string, payload string, callback SendReqCallback, data []byte) {
	replyLevel := packet.RLevelReplyLater
	if callback == nil {
		replyLevel = packet.RLevelNoReply
	}
	client.msgIdLock.Lock()
	msgId := client.reqMsgId
	client.reqMsgId++
	client.msgIdLock.Unlock()
	//If there is a callback for the request, save it into a different map
	//如果回调不为空，加入等待回复的消息map
	if callback != nil {
		client.mapLock.Lock()
		client.reqMsgMap[msgId] = callback
		client.mapLock.Unlock()
	}
	hasData := false
	if len(data) > 0 {
		hasData = true
	}
	sendReqMsg := &packet.SendReq{
		MessageId:  msgId,
		ReplyLevel: replyLevel,
		Type:       payloadType,
		Payload:    payload,
		Data:       data,
		HasData:    hasData,
	}
	client.sync(sendReqMsg)
}

func (client *SocketClientConn) SendPing() {
	pingMsg := &packet.PingReq{}
	client.sync(pingMsg)
	//log.Println("ping sent")
}

func (client *SocketClientConn) handleSendReq(msgType string, msgPayload string) {
	defer func() {
		if err := recover(); err != nil {
			client.log.Error(err)
		}
	}()
	if client.cInterface != nil {
		client.cInterface.OnSendReqReceived(msgType, msgPayload)
	}
}

func (client *SocketClientConn) handleSendResp(msgId uint16, msgPayload string) {
	client.mapLock.RLock()
	callback := client.reqMsgMap[msgId]
	client.mapLock.RUnlock()
	//如果有回调
	if callback != nil {
		//Run asynchronously, make sure the callback won't block the message handling
		//异步执行，确保回调不会卡消息处理
		go func() {
			callback(msgPayload)
		}()
		client.mapLock.Lock()
		delete(client.reqMsgMap, msgId)
		client.mapLock.Unlock()
	}
}

// Add the message into sending queue and wait synchronously until the message is sent
// 将消息同步加入任务队列，阻塞直到消息发送完成
func (client *SocketClientConn) sync(message packet.IMessage) {
	defer func() {
		if err := recover(); err != nil {
			client.log.Error(err)
		}
	}()
	job := Job{
		Message: message,
		Receipt: make(Receipt),
	}
	client.jobChan <- job
	//Block until the message is sent
	//阻塞直到消息发送完成
	job.Receipt.Wait()
}

// Add the message into sending queue and return immediately
// 将消息异步加入任务队列
func (client *SocketClientConn) submit(message packet.IMessage) {
	job := Job{
		Message: message,
	}
	client.jobChan <- job
}
