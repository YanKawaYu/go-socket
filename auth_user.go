package gosocket

import (
	"github.com/yankawayu/go-socket/packet"
	"go.uber.org/zap/zapcore"
)

type IUser interface {
	Auth(payload string, ip string) (uid int64, code packet.ReturnCode) //获取用户信息
	Login(uid int64) packet.ReturnCode                                  //登陆
	Logout(isKickOut bool)                                              //注销
	Refresh()                                                           //每隔一段时间，更新在线状态
	IsLogin() bool                                                      //是否登陆
	RequireLock(uid int64) bool                                         //获取用户状态锁
	ReleaseLock(uid int64)                                              //释放用户状态锁

	GetUid() int64                   //用户id
	GetConnectInfo() []zapcore.Field //连接信息
	GetSendReqInfo() []zapcore.Field //请求信息

	// HandleNoReplyReq 处理不需要回复的SendReq，目前不需要回复的消息都是修改用户状态，故暂时放在User中
	HandleNoReplyReq(payloadType string, payload string)
}

// AuthUser 默认登陆验证父类，继承后实现具体登陆逻辑
// Default login auth class, should inherit this class to implement concrete auth logic
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
