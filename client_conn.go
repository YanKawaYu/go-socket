package gosocket

import (
	"github.com/pkg/errors"
	"github.com/yankawayu/go-socket/packet"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// kQueueLength is used to set the max amount of jobs waiting to be sent.
// This limit is to avoid running out of memory in extreme cases
// 任务队列的长度
const kQueueLength = 200

// Receipt is used to get notified when the message is sent
// 用于任务完成的通知
type Receipt chan struct{}

func (receipt Receipt) Wait() {
	// TODO: timeout
	<-receipt
}

// Job is used to store the message that is about to be sent out
type Job struct {
	Message packet.IMessage // Message is about to be sent
	Receipt Receipt         // Receipt is used to get notified when the message is sent
}

// ClientConn is used to handle individual valid connection coming to the server
// It contains three threads including Reading thread, writing thread and handling thread
// Reading thread is used to read and decode input data into messages, then put those messages into Handling thread
// Writing thread is used to send all the messages back as output data
// Handling thread is used to process all the incoming messages from Reading thread and generate response messages
type ClientConn struct {
	conn       net.Conn
	clientIp   string                 //Used to store the ip address of the client
	jobChan    chan Job               //Used to store all the jobs that are about to be sent
	handler    *MessageHandler        //Used to handle all the messages coming in
	msgManager *packet.MessageManager //Used to help encoding and decoding messages
}

func NewClientConn(conn net.Conn) (client *ClientConn) {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
			client = nil
		}
	}()
	clientIp := conn.RemoteAddr().String()
	//Get ip only
	//忽略端口号，只取ip
	ipAndPort := strings.Split(clientIp, ":")
	if len(ipAndPort) > 0 {
		clientIp = ipAndPort[0]
	}
	jobChan := make(chan Job, kQueueLength)
	client = &ClientConn{
		conn:       conn,
		clientIp:   clientIp,
		jobChan:    jobChan,
		handler:    NewMessageHandler(jobChan, clientIp),
		msgManager: &packet.MessageManager{},
	}
	return
}

func (client *ClientConn) Start() {
	//Reading thread
	//读线程
	go client.startReader()
	//Writing thread
	//写线程
	go client.startWriter()
	//Handling thread
	//处理消息
	go client.handler.Start()
}

// 开启写线程
func (client *ClientConn) startWriter() {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
		}
	}()
	//This is a new defer block.
	//It's done on purpose to make sure that panic in this block is catchable by the first defer block
	//新开一个defer，确保这个defer中有panic也能被捕捉到
	defer func() {
		client.conn.Close()
	}()
	for job := range client.jobChan {
		err := client.msgManager.EncodeMessage(client.conn, job.Message)
		//Notify the job is done (the message is sent)
		//通知消息发送完成
		if job.Receipt != nil {
			close(job.Receipt)
		}

		if err != nil {
			//Network error in tls connection
			//tls中断连接的错误
			if strings.HasSuffix(err.Error(), "use of closed connection") {
				return
			}
			//Network error in non-tls connection
			//普通连接中断的错误
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}
			TcpApp.Log.Error(err)
			return
		}
		//If the job just sent is Disconnect message, stop the Writing Thread immediately
		//断开连接后确保马上返回
		if _, ok := job.Message.(*packet.Disconnect); ok {
			return
		}
	}
}

// 开启读线程
func (client *ClientConn) startReader() {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
		}
	}()
	//This is a new defer block.
	//It's done on purpose to make sure that panic in this block is catchable by the first defer block
	//新开一个defer，确保这个defer中有panic也能被捕捉到
	defer func() {
		client.conn.Close()
		//Stop the handling thread
		//If the connection was kick out by the same user, then the handling thread should be stopped already.
		//In this situation, this call is useless
		//如果是被同一账号踢出，之前已经调用过stop，这次调用没什么作用
		client.handler.Stop(false)
	}()
	for {
		//The read timeout should be 1.5 times bigger than the interval of the ping pong message
		//To avoid the network being on and off
		//超时时间，设置为心跳包间隔的1.5倍，避免复杂网络
		timeoutInterval := time.Duration(float64(client.msgManager.ProCommon.KeepAliveTime) * 1.5)
		if timeoutInterval > 0 {
			client.conn.SetReadDeadline(time.Now().Add(timeoutInterval * time.Second))
		}
		//Get the message
		//获取消息
		msg, err := client.msgManager.DecodeMessage(client.conn)
		if err != nil {
			if err == io.EOF {
				//If the client close the connection from the other side
				//客户端关闭连接
				return
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				//If the connection is idle without any data including ping pong messages
				TcpApp.Log.Debugf("user %d client conn timeout", client.handler.user.GetUid())
				return
			}
			//Network error in tls connection
			//tls中断连接的错误
			if strings.HasSuffix(err.Error(), "use of closed connection") {
				return
			}
			//Network error in non-tls connection
			//普通连接中断的错误
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}
			//Client cut the connection
			//客户端中断连接的错误
			if strings.HasSuffix(err.Error(), "connection reset by peer") {
				return
			}
			//Use a non-tls client to connect a tls server
			//非TLS连接的错误
			if strings.HasSuffix(err.Error(), "first record does not look like a TLS handshake") {
				return
			}
			//Errors regarding gzip
			//gzip错误
			if strings.HasPrefix(err.Error(), "gzip") {
				return
			}
			//Other tls error
			//tls其他错误
			if strings.HasPrefix(err.Error(), "tls") {
				return
			}
			//Log all error messages under debug environment
			//如果是自己定义的消息错误，仅在Debug环境输出
			if _, ok := err.(packet.MessageErr); ok {
				TcpApp.Log.Debug(errors.Wrap(err, client.clientIp))
			} else {
				TcpApp.Log.Error(errors.Wrap(err, client.clientIp))
			}
			return
		}
		select {
		case client.handler.workChan <- msg:
		default:
			TcpApp.Log.Warning(strconv.FormatInt(client.handler.user.GetUid(), 10) + " fail to add message: " + JSONEncode(msg))
		}
	}
}
