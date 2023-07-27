package tcpclient

import (
	"github.com/yankawayu/go-socket"
	"github.com/yankawayu/go-socket/packet"
	"github.com/yankawayu/go-socket/utils"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

const QueueLength = 50

type SendReqCallback func(payloadBody string)

type ClientConnInterface interface {
	//监听服务器推送
	OnSendReqReceived(reqType string, reqBody string)
	OnDisconnect()
	//解析服务器返回
	DecodeResponse(payloadBody string) (error, string)
}

type ClientConn struct {
	cInterface	ClientConnInterface
	conn		net.Conn
	jobChan		chan gotcp.Job        //任务队列
	connAckChan	chan *packet.ConnAck      //连接回复队列
	reqMsgId	uint16                       //协议自增消息id
	msgIdLock	*sync.RWMutex
	reqMsgMap	map[uint16]SendReqCallback //等待回复的消息map
	mapLock		*sync.RWMutex

	msgManager	*packet.MessageManager //协议层的包管理器
	log			utils.ILogger //输出日志用
}

func NewClientConn(connection net.Conn, log utils.ILogger) *ClientConn {
	cli := &ClientConn{
		conn:		connection,
		jobChan:	make(chan gotcp.Job, QueueLength),
		connAckChan:	make(chan *packet.ConnAck),
		reqMsgId:	1,
		msgIdLock: &sync.RWMutex{},
		reqMsgMap:	make(map[uint16]SendReqCallback),
		mapLock: &sync.RWMutex{},

		msgManager: &packet.MessageManager{
			ProCommon: packet.ProtocolCommon{
				ProName: packet.ProtocolName, //协议名
				ProVersion: packet.ProtocolVersion, //协议版本号
				KeepAliveTime: 60, //心跳包间隔
				EnablePayloadGzip: true, //是否开启gzip
			},
		},
		log: log,
	}
	go cli.startReader()
	go cli.startWriter()
	return cli
}

func (client *ClientConn) SetConnInterface(connInterface ClientConnInterface)  {
	client.cInterface = connInterface
}

func (client *ClientConn) GetConnInterface() ClientConnInterface {
	return client.cInterface
}

func (client *ClientConn) startReader() {
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

func (client *ClientConn) startWriter() {
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

func (client *ClientConn) Connect(loginInfo string) error {
	connectMsg := &packet.Connect{
		Payload:         loginInfo,
	}
	//将消息加入任务队列，阻塞直到消息发送完成
	client.sync(connectMsg)
	//阻塞等待连接回复
	ack := <-client.connAckChan
	return packet.ConnectionErrors[ack.ReturnCode]
}

func (client *ClientConn) Disconnect() {
	disconnectMsg := &packet.Disconnect{}
	client.submit(disconnectMsg)
}

func (client *ClientConn) SendRequest(payloadType string, payload string, callback SendReqCallback, data []byte) {
	replyLevel := packet.RLevelReplyLater
	if callback == nil {
		replyLevel = packet.RLevelNoReply
	}
	client.msgIdLock.Lock()
	msgId := client.reqMsgId
	client.reqMsgId++
	client.msgIdLock.Unlock()
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
	//协议包
	sendReqMsg := &packet.SendReq{
		MessageId:  msgId,
		ReplyLevel: replyLevel,
		Type:	    payloadType,
		Payload:    payload,
		Data:		data,
		HasData: 	hasData,
	}
	client.sync(sendReqMsg)
}

func (client *ClientConn) SendPing() {
	pingMsg := &packet.PingReq{}
	client.sync(pingMsg)
	//log.Println("ping sent")
}

func (client *ClientConn) handleSendReq(msgType string, msgPayload string) {
	defer func() {
		if err := recover(); err != nil {
			client.log.Error(err)
		}
	}()
	if client.cInterface != nil {
		client.cInterface.OnSendReqReceived(msgType, msgPayload)
	}
}

func (client *ClientConn) handleSendResp(msgId uint16, msgPayload string)  {
	client.mapLock.RLock()
	callback := client.reqMsgMap[msgId]
	client.mapLock.RUnlock()
	//如果有回调
	if callback != nil {
		//异步执行，确保回调不会卡消息处理
		go func() {
			callback(msgPayload)
		}()
		client.mapLock.Lock()
		delete(client.reqMsgMap, msgId)
		client.mapLock.Unlock()
	}
}

//将消息加入任务队列，阻塞直到消息发送完成
func (client *ClientConn) sync(message packet.IMessage) {
	defer func() {
		if err := recover(); err != nil {
			client.log.Error(err)
		}
	}()
	job := gotcp.Job{
		Message: message,
		Receipt: make(gotcp.Receipt),
	}
	//加入任务队列
	client.jobChan <- job
	//阻塞直到消息发送完成
	job.Receipt.Wait()
}

func (client *ClientConn) submit(message packet.IMessage) {
	job := gotcp.Job{
		Message: message,
	}
	client.jobChan <- job
}