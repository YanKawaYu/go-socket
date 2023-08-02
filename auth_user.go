package gosocket

import (
	"github.com/yankawayu/go-socket/packet"
	"go.uber.org/zap/zapcore"
)

type IUser interface {
	// Auth Get user information from the connect info
	// 获取用户信息
	Auth(payload string, ip string) (uid int64, code packet.ReturnCode)
	// Login Change the user status on the server, mark the user is online
	// 登陆
	Login(uid int64) packet.ReturnCode
	// Logout Change the user status on the server, mark the user is offline
	// 注销
	Logout(isKickOut bool)
	// Refresh the user online status
	//每隔一段时间，更新在线状态
	Refresh()
	// IsLogin check whether the user is online
	// 是否登陆
	IsLogin() bool
	// RequireLock get a lock before change the user's online status to prevent concurrent login
	// 获取用户状态锁
	RequireLock(uid int64) bool
	// ReleaseLock release the lock
	// 释放用户状态锁
	ReleaseLock(uid int64)
	// GetUid get the current user's id
	// 用户id
	GetUid() int64
	// GetConnectInfo log extra connect info
	// 需要额外记录进日志的连接信息
	GetConnectInfo() []zapcore.Field
	// GetSendReqInfo log extra request info
	// 需要额外记录进日志的请求信息
	GetSendReqInfo() []zapcore.Field
	// HandleNoReplyReq handle all the requests that don't require response
	// For now all this kind of requests is modifying user status, that's why this function is here in User
	// 处理不需要回复的SendReq，目前不需要回复的消息都是修改用户状态，故暂时放在User中
	HandleNoReplyReq(payloadType string, payload string)
}

// AuthUser Default login auth class, should inherit this class to implement concrete auth logic
// 默认登陆验证父类，继承后实现具体登陆逻辑
type AuthUser struct {
	Uid int64
}

func (user *AuthUser) Refresh() {}

func (user *AuthUser) GetUid() int64 {
	return user.Uid
}

func (user *AuthUser) GetConnectInfo() []zapcore.Field {
	return []zapcore.Field{}
}

func (user *AuthUser) GetSendReqInfo() []zapcore.Field {
	return []zapcore.Field{}
}

func (user *AuthUser) HandleNoReplyReq(payloadType string, payload string) {}

func (user *AuthUser) IsLogin() bool {
	return user.Uid != 0
}

func (user *AuthUser) RequireLock(uid int64) bool {
	return true
}

func (user *AuthUser) ReleaseLock(uid int64) {}

func (user *AuthUser) Auth(payload string, ip string) (uid int64, code packet.ReturnCode) {
	return -1, packet.RetCodeAccepted
}

func (user *AuthUser) Login(uid int64) packet.ReturnCode {
	user.Uid = uid
	return packet.RetCodeAccepted
}

func (user *AuthUser) Logout(isKickOut bool) {
	user.Uid = 0
}
