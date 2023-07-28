package gosocket

import (
	"encoding/json"
	"fmt"
	"github.com/yankawayu/go-socket/packet"
	"github.com/yankawayu/go-socket/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"reflect"
	"time"
)

var (
	authUser IUser
)

const (
	kAccessLogType     = "type"
	kAccessLogIp       = "ip"
	kAccessLogUid      = "uid"
	kAccessLogParams   = "params"
	kAccessLogStatus   = "status"
	kAccessLogMessage  = "message"
	kAccessLogDuration = "duration"
)

// 设置登陆验证类
func setAuthUser(user IUser) {
	authUser = user
}

type MessageHandler struct {
	user     IUser                //用户信息
	jobChan  chan Job             //发出消息任务队列
	workChan chan packet.IMessage //收到消息任务队列
	ip       string               //客户端ip
	isStop   bool                 //是否停止
}

func NewMessageHandler(jobChan chan Job, ip string) *MessageHandler {
	handler := &MessageHandler{
		jobChan:  jobChan,
		workChan: make(chan packet.IMessage, kQueueLength),
		ip:       ip,
		isStop:   false,
	}
	//验证
	userReflectVal := reflect.ValueOf(authUser)
	userType := reflect.Indirect(userReflectVal).Type()
	//获取user
	user := reflect.New(userType)
	execUser, ok := user.Interface().(IUser)
	if !ok {
		TcpApp.Log.Error("controller is not IController")
	}
	handler.user = execUser
	return handler
}

// Start 开始处理消息
// start to handle messages
func (handler *MessageHandler) Start() {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
		}
	}()
	defer func() {
		//关闭发消息任务队列
		if handler.jobChan != nil {
			close(handler.jobChan)
			handler.jobChan = nil
		}
	}()
	//维持在线状态的时间
	refreshTime := time.Now()
	for {
		if handler.workChan == nil {
			return
		}
		select {
		case msg, isOpen := <-handler.workChan:
			if !isOpen {
				return
			}
			switch msg := msg.(type) {
			case *packet.Connect:
				if handler.handleConnect(msg) {

				} else {
					//登陆失败，跳出循环
					return
				}
			case *packet.SendReq:
				if handler.user.IsLogin() {
					handler.handleSendReq(msg)
				} else {
					//TcpApp.Log.Debug("ping request receive without login, disconnect...")
					return
				}
			case *packet.PingReq:
				if handler.user.IsLogin() {
					handler.handlePingReq(msg)
				} else {
					//TcpApp.Log.Debug("ping request receive without login, disconnect...")
					return
				}
			case *packet.Disconnect:
				//断开连接
				//TcpApp.Log.Debug("disconnect received")
				return
			case *packet.ConnAck, *packet.PingResp, *packet.SendResp:
				//服务器不应该收到的消息类型，断开连接
				TcpApp.Log.Debug("invalid message type, disconnect")
				return
			default:
				//未知消息类型
				TcpApp.Log.Debug("read unknown message type, disconnect...", msg)
				return
			}
			//如果已登陆
			if handler.user.IsLogin() {
				//距离上一次刷新超过3分钟，在线状态的过期时间必须大于3+1分钟，目前状态过期时间是5分钟
				if time.Now().Sub(refreshTime) > 3*time.Minute {
					//刷新在线状态
					handler.user.Refresh()
					//更新刷新时间
					refreshTime = time.Now()
				}
			}
		case <-time.After(time.Second):
			break
		}
		// 判断. 若需要退出, 且此时读写队列都没有数据了, 则断开链接
		if utils.GetRestartManager().IsStop() &&
			len(handler.jobChan) <= 0 &&
			len(handler.workChan) <= 0 {
			break
		}
	}
}

// Stop 停止处理消息
// stop to handle messages
func (handler *MessageHandler) Stop(isKickOut bool) {
	//如果停止过了，直接返回，避免stop在短时间内两次调用导致用户在线状态被清空
	if handler.isStop {
		return
	}
	handler.isStop = true
	//如果工作队列未关闭，关闭
	if handler.workChan != nil {
		close(handler.workChan)
		handler.workChan = nil
	}
	//如果已登陆，注销
	if handler.user.IsLogin() {
		//如果不是被同一账号登陆踢出
		//否则可能会移除掉最新登陆的状态
		if !isKickOut {
			//移除本地记录的在线状态
			GetClientPool().RemoveClientByUid(handler.user.GetUid())
		}
		handler.user.Logout(isKickOut)
	}
}

// 连接消息
func (handler *MessageHandler) handleConnect(msg *packet.Connect) (isConnect bool) {
	startTime := time.Now()
	var returnCode packet.ReturnCode
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
			returnCode = packet.RetCodeServerUnavailable
		}
		//发送连接回执
		msgConnAck := &packet.ConnAck{
			ReturnCode: returnCode,
		}
		handler.submitSync(msgConnAck)
		//返回是否连接成功
		isConnect = returnCode == packet.RetCodeAccepted
		//处理时间
		processDuration := fmt.Sprintf("%.3f", float32(time.Now().Sub(startTime))/float32(time.Second))
		//记录日志
		message := ""
		if returnCode != packet.RetCodeAccepted {
			message = packet.ConnectionErrors[returnCode].Error()
		}
		//连接信息
		connectInfo := []zapcore.Field{
			zap.String(kAccessLogIp, handler.ip),
			zap.Int64(kAccessLogUid, handler.user.GetUid()),
			zap.String(kAccessLogParams, msg.Payload),
			zap.Uint8(kAccessLogStatus, uint8(returnCode)),
			zap.String(kAccessLogMessage, message),
			zap.String(kAccessLogDuration, processDuration),
		}
		//添加自定义连接信息
		connectInfo = append(connectInfo, handler.user.GetConnectInfo()...)
		TcpApp.FastLog.Info("connect", connectInfo...)
	}()
	//获取用户信息
	var uid int64
	uid, returnCode = handler.user.Auth(msg.Payload, handler.ip)
	if returnCode == packet.RetCodeAccepted && uid != 0 {
		//获取锁
		hasLock := handler.user.RequireLock(uid)
		if hasLock {
			//验证登陆信息
			returnCode = handler.user.Login(uid)
			//如果登陆成功，在当前服务器上记录在线状态
			if returnCode == packet.RetCodeAccepted {
				//如果之前连接过，说明在新旧连接在同一台服务器上，需要将旧的连接移除
				oldHandler := GetClientPool().GetClientByUid(handler.user.GetUid())
				if oldHandler != nil {
					//通知客户端连接断开
					msgDisconnect := &packet.Disconnect{
						Type: packet.DiscTypeKickout,
					}
					oldHandler.Submit(msgDisconnect)
					//停止处理消息
					oldHandler.Stop(true)
					TcpApp.Log.Debugf("kick out same server account %d", uid)
				}
				//设置最新的在线状态
				GetClientPool().SetClientByUid(handler, handler.user.GetUid())
			}
			//释放锁
			handler.user.ReleaseLock(uid)
		} else {
			returnCode = packet.RetCodeConcurrentLogin
		}
	}
	return
}

// 请求消息
func (handler *MessageHandler) handleSendReq(msg *packet.SendReq) {
	defer func() {
		if err := recover(); err != nil {
			TcpApp.Log.Error(err)
		}
	}()
	switch msg.ReplyLevel {
	case packet.RLevelNoReply:
		startTime := time.Now()
		//处理不需要回复的消息
		handler.user.HandleNoReplyReq(msg.Type, msg.Payload)
		//处理时间
		processDuration := fmt.Sprintf("%.3f", float32(time.Now().Sub(startTime))/float32(time.Second))
		//记录日志
		sendReqInfo := []zapcore.Field{
			zap.String(kAccessLogType, msg.Type),
			zap.String(kAccessLogIp, handler.ip),
			zap.Int64(kAccessLogUid, handler.user.GetUid()),
			zap.String(kAccessLogParams, msg.Payload),
			zap.String(kAccessLogDuration, processDuration),
		}
		//添加自定义请求信息
		sendReqInfo = append(sendReqInfo, handler.user.GetSendReqInfo()...)
		TcpApp.FastLog.Info("sendReqNoReply", sendReqInfo...)
	case packet.RLevelReplyLater:
		startTime := time.Now()
		//业务逻辑
		response := ProcessPayloadWithData(handler.user, msg.Type, msg.Payload, msg.Data)
		//处理时间
		processDuration := fmt.Sprintf("%.3f", float32(time.Now().Sub(startTime))/float32(time.Second))
		//检查参数中是否有过长的
		var paramMap map[string]json.RawMessage
		err := json.Unmarshal([]byte(msg.Payload), &paramMap)
		tmpMap := map[string]json.RawMessage{}
		if err == nil {
			for k, v := range paramMap {
				//过滤过长的参数，避免图片这种导致日志过多
				if len(v) > 50 {
					continue
				}
				tmpMap[k] = v
			}
		}
		//记录日志
		sendReqInfo := []zapcore.Field{
			zap.String(kAccessLogType, msg.Type),
			zap.String(kAccessLogIp, handler.ip),
			zap.Int64(kAccessLogUid, handler.user.GetUid()),
			zap.Any(kAccessLogParams, tmpMap),
			zap.Uint8(kAccessLogStatus, uint8(response.Status)),
			zap.String(kAccessLogMessage, response.Message),
			zap.String(kAccessLogDuration, processDuration),
		}
		//添加自定义请求信息
		sendReqInfo = append(sendReqInfo, handler.user.GetSendReqInfo()...)
		TcpApp.FastLog.Info("sendReq", sendReqInfo...)
		//答复结果
		sendResp := &packet.SendResp{
			MessageId: msg.MessageId,
			Payload:   JSONEncode(response),
		}
		handler.Submit(sendResp)
	}
}

// 心跳消息
func (handler *MessageHandler) handlePingReq(msg *packet.PingReq) {
	pingResp := &packet.PingResp{}
	handler.Submit(pingResp)
}

// PushNotify 发推送到客户端
// Send push message to the client
func (handler *MessageHandler) PushNotify(notifyType string, body interface{}) {
	msgReq := &packet.SendReq{
		Type:       notifyType,
		Payload:    JSONEncode(body),
		ReplyLevel: packet.RLevelNoReply,
	}
	handler.Submit(msgReq)
}

// Submit 发送消息，异步进行，消息发送成功就返回，如果任务队列满了则忽略消息
// Send message asynchronously
func (handler *MessageHandler) Submit(message packet.IMessage) {
	job := Job{
		Message: message,
	}
	//任务队列未被关闭
	if handler.jobChan != nil {
		select {
		case handler.jobChan <- job:
		default:
			fullMessage := fmt.Sprintf("%d's job queue full", handler.user.GetUid())
			TcpApp.Log.Error(fullMessage)
		}
	}
	return
}

// 发送消息，同步进行，只有消息发送成功且被处理完之后才返回，如果队列满了则等待
func (handler *MessageHandler) submitSync(message packet.IMessage) {
	job := Job{
		Message: message,
		Receipt: make(Receipt),
	}
	//任务队列未被关闭
	if handler.jobChan != nil {
		//加入任务队列，必须判断channel是否满，否则会死锁
		select {
		case handler.jobChan <- job:
			//阻塞直到消息发送完成
			job.Receipt.Wait()
		default:
			fullMessage := fmt.Sprintf("%d's job queue full", handler.user.GetUid())
			TcpApp.Log.Error(fullMessage)
		}
	}
	return
}
