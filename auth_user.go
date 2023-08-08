package gosocket

import (
	"github.com/yankawayu/go-socket/packet"
	"go.uber.org/zap/zapcore"
)

type IUser interface {
	// Auth Check whether the login information provided by the client is valid
	// 获取用户信息
	Auth(payload string, ip string) (uid int64, code packet.ReturnCode)
	// Login Change the user status on the server, mark the user is online
	// Return RetCodeAccepted to proceed the login process, otherwise to close this connection
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

// Refresh Override this function to refresh the online status of the user
// This function will be called every 3 minutes as long as the connection is still on
// It is recommended that you can use REDIS to maintain a global online status for all users across all servers
// REDIS is an open source (BSD licensed), in-memory data structure store. It can be used as a cache database
//
// You can save the user's online status with an expiry time(e.g. 5min) in REDIS
// Then you can implement this function to reset the expiry time so that the user will be marked online all the time
//
// If you plan to deploy only one server, you can choose go map
func (user *AuthUser) Refresh() {}

// GetUid This function is rarely changed
func (user *AuthUser) GetUid() int64 {
	return user.Uid
}

// GetConnectInfo Override this function to put extra connect info into the log
// For example, device info, network info, system info etc.
func (user *AuthUser) GetConnectInfo() []zapcore.Field {
	return []zapcore.Field{}
}

// GetSendReqInfo Override this function to put extra request info into the log
// For example, device info, network info, system info etc.
func (user *AuthUser) GetSendReqInfo() []zapcore.Field {
	return []zapcore.Field{}
}

// HandleNoReplyReq Override this function to handle the requests that don't need to be responded
// These kinds of requests are normally insignificance.
// Even if they got lost in the network, it doesn't matter.
// Cutting off these kinds of response will be an improvement to the crowded network
func (user *AuthUser) HandleNoReplyReq(payloadType string, payload string) {}

// IsLogin This function is rarely changed
func (user *AuthUser) IsLogin() bool {
	return user.Uid != 0
}

// RequireLock Override this function to get a lock before operating the user's data
// As the login process `handleConnect` in MessageHandler
// We need locks here to avoid the situation of same account trying to log in from different devices simultaneously
//
// It is recommended to use REDIS to record different lock for different user across different servers
// If you plan to deploy only one server, you can choose sync.Mutex
func (user *AuthUser) RequireLock(uid int64) bool {
	return true
}

// ReleaseLock Implement this function to release the lock after operating the user's data
// This function goes with RequireLock
func (user *AuthUser) ReleaseLock(uid int64) {}

// Auth Override this function to validate the user's login information
// It is recommended to use payload to transmit auth token instead of password md5
//
// If the login information is valid, then return the user's uid and packet.RetCodeAccepted
// If the login information is invalid, then return an invalid uid and packet.RetCodeBadLoginInfo
// For more code, see packet.ReturnCode
// It is recommended to use -1 as invalid uid, since 0 is used as not login
func (user *AuthUser) Auth(payload string, ip string) (uid int64, code packet.ReturnCode) {
	return -1, packet.RetCodeAccepted
}

// Login Override this function to record user's online status
//
// If you have multiple servers, you need to record which server the user is on
// And for the future real-time notification you need to make sure the user is currently connected to only one server
//
// You should return packet.RetCodeAccepted if everything goes well
// Or else you can return packet.RetCodeBadLoginInfo to block the login process
func (user *AuthUser) Login(uid int64) packet.ReturnCode {
	user.Uid = uid
	return packet.RetCodeAccepted
}

// Logout Override this function to mark the user is offline
// The param `isKickOut` is used to mark whether the user is kicked out by the user himself
// If the user reconnect on the same server, it will cause the old connection to be kicked out (It usually happens under bad network)
func (user *AuthUser) Logout(isKickOut bool) {
	user.Uid = 0
}
