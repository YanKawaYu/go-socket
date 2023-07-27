package gotcp

import (
	"github.com/pkg/errors"
	"github.com/yankawayu/go-socket/packet"
	"go.uber.org/zap/zapcore"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

//任务队列的长度
const kQueueLength = 200

//用于任务完成的通知
type Receipt chan struct{}

func (receipt Receipt) Wait() {
	// TODO: timeout
	<-receipt
}

type Job struct {
	Message packet.IMessage
	Receipt Receipt
}

type IUser interface {
	Auth(payload string, ip string) (uid int64, code packet.ReturnCode) //获取用户信息
	Login(uid int64) packet.ReturnCode //登陆
	Logout(isKickOut bool) //注销
	Refresh() //每隔一段时间，更新在线状态
	IsLogin() bool //是否登陆
	RequireLock(uid int64) bool //获取用户状态锁
	ReleaseLock(uid int64) //释放用户状态锁

	GetUid() int64 //用户id
	GetConnectInfo() []zapcore.Field //连接信息
	GetSendReqInfo() []zapcore.Field //请求信息

	//处理不需要回复的SendReq，目前不需要回复的消息都是修改用户状态，故暂时放在User中
	HandleNoReplyReq(payloadType string, payload string)
}

type ClientConn struct {
	conn		net.Conn
	clientIp	string
	jobChan 	chan Job		//发出消息任务队列
	handler		*MessageHandler		//消息处理
	msgManager	*packet.MessageManager
}

func NewClientConn(conn net.Conn) (client *ClientConn) {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
			client = nil
		}
	}()
	clientIp := conn.RemoteAddr().String()
	//忽略端口号，只取ip
	ipAndPort := strings.Split(clientIp, ":")
	if len(ipAndPort)>0 {
		clientIp = ipAndPort[0]
	}
	jobChan := make(chan Job, kQueueLength)
	client = &ClientConn{
		conn:		conn,
		clientIp:	clientIp,
		jobChan:	jobChan,
		handler: 	NewMessageHandler(jobChan, clientIp),
		msgManager: &packet.MessageManager{},
	}
	return
}

func (client *ClientConn) Start ()  {
	//读线程
	go client.startReader()
	//写线程
	go client.startWriter()
	//处理消息
	go client.handler.Start()
}

//开启写线程
func (client *ClientConn) startWriter() {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
		}
	}()
	//新开一个defer，确保这个defer中有panic也能被捕捉到
	defer func() {
		client.conn.Close()
	}()
	for job := range client.jobChan {
		err := client.msgManager.EncodeMessage(client.conn, job.Message)
		//通知消息发送完成
		if job.Receipt != nil {
			close(job.Receipt)
		}
		//如果出错
		if err != nil {
			//tls中断连接的错误
			if strings.HasSuffix(err.Error(), "use of closed connection") {
				return
			}
			//普通连接中断的错误
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}
			TcpApp.Log.Error(err)
			return
		}
		//断开连接后确保马上返回
		if _, ok := job.Message.(*packet.Disconnect); ok {
			return
		}
	}
}

//开启读线程
func (client *ClientConn) startReader() {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
		}
	}()
	//新开一个defer，确保这个defer中有panic也能被捕捉到
	defer func() {
		client.conn.Close()
		//停止处理消息
		//如果是被同一账号踢出，之前已经调用过stop，这次调用没什么作用
		client.handler.Stop(false)
	}()
	for {
		//超时时间，设置为心跳包间隔的1.5倍，避免复杂网络
		timeoutInterval := time.Duration(float64(client.msgManager.ProCommon.KeepAliveTime)*1.5)
		if timeoutInterval > 0 {
			client.conn.SetReadDeadline(time.Now().Add(timeoutInterval*time.Second))
		}
		//获取消息
		msg, err := client.msgManager.DecodeMessage(client.conn)
		if err != nil {
			if err == io.EOF {
				//客户端关闭连接
				return
			}else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				//用于调试用户连接是否断开
				TcpApp.Log.Debugf("user %d client conn timeout", client.handler.user.GetUid())
				//超时，长时间没有收到心跳包
				return
			}
			//tls中断连接的错误
			if strings.HasSuffix(err.Error(), "use of closed connection") {
				return
			}
			//普通连接中断的错误
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}
			//客户端中断连接的错误
			if strings.HasSuffix(err.Error(), "connection reset by peer") {
				return
			}
			//非TLS连接的错误
			if strings.HasSuffix(err.Error(), "first record does not look like a TLS handshake") {
				return
			}
			//gzip错误
			if strings.HasPrefix(err.Error(), "gzip") {
				return
			}
			//tls其他错误
			if strings.HasPrefix(err.Error(), "tls") {
				return
			}
			//如果是自己定义的消息错误，仅在Debug环境输出
			if _, ok := err.(packet.MessageErr); ok {
				TcpApp.Log.Debug(errors.Wrap(err, client.clientIp))
			}else {
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
