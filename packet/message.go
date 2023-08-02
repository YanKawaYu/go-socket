package packet

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io"
)

const ProtocolName = "GOSOC"
const ProtocolVersion = 1

// 消息类型
const (
	MsgConnect = MessageType(iota + 1)
	MsgConnAck
	MsgPingReq
	MsgPingResp
	MsgDisconnect
	MsgSendReq  //客户端或服务器发消息
	MsgSendResp //客户端或服务器回复消息

	msgTypeFirstInvalid
)

type MessageType uint8

func (mt MessageType) IsValid() bool {
	return mt >= MsgConnect && mt < msgTypeFirstInvalid
}

// 报文最大长度
const (
	// Maximum payload size in bytes 256MB
	MaxPayloadSize = (1 << (4 * 7)) - 1
)

type IMessage interface {
	// 将消息输出到写通道
	Encode(writer io.Writer, proCommon *ProtocolCommon) error
	// 从读通道读出消息
	Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) error
}

// 写入消息的头部
func writeMessageHeader(headerBuf *bytes.Buffer, fixHeader *FixHeader, mutableHeader *bytes.Buffer, payloadLen int32) error {
	var mutableHeaderLen int64
	//可变头部是否为空
	if mutableHeader != nil {
		mutableHeaderLen = int64(len(mutableHeader.Bytes()))
	} else {
		mutableHeaderLen = 0
	}
	//计算报文剩余总长度（可变报头+有效载荷）
	totalPayloadLength := mutableHeaderLen + int64(payloadLen)
	//如果报文剩余总长度大于最大长度
	if totalPayloadLength > MaxPayloadSize {
		return NewMessageError(fmt.Sprintf(msgTooLongError+" beyond max:%d", totalPayloadLength))
	}
	fixHeader.remainLen = int32(totalPayloadLength)

	buf := new(bytes.Buffer)
	//向缓冲写入消息固定头部
	err := fixHeader.EncodeInto(buf)
	if err != nil {
		return err
	}
	//如果有可变头部，向缓冲写入消息可变头部
	if mutableHeader != nil {
		buf.Write(mutableHeader.Bytes())
	}
	//将缓冲发出
	_, err = headerBuf.Write(buf.Bytes())
	return err
}

/**
 *	连接消息
 */
type Connect struct {
	header          FixHeader //固定头部
	protocolName    string    //协议名，DPIM
	protocolVersion uint8     //1开始
	//flags           uint8     //是否采用gzip等
	keepAliveTime uint16 //连接间隔时间
	Payload       string //JSON

	enablePayloadGzip bool //包含在flags中
}

func (msg *Connect) Encode(writer io.Writer, proCommon *ProtocolCommon) (err error) {
	msg.header.MsgType = MsgConnect

	buf := new(bytes.Buffer)
	//协议名
	setString(msg.protocolName, buf)
	//协议版本
	setUint8(msg.protocolVersion, buf)
	//标志位，预留
	flags := boolToByte(msg.enablePayloadGzip) << 7
	buf.WriteByte(flags)
	//连接时间
	setUint16(msg.keepAliveTime, buf)
	//初始化载荷
	payloadBuf := new(bytes.Buffer)
	if msg.enablePayloadGzip {
		setGzipString(msg.Payload, payloadBuf)
	} else {
		setString(msg.Payload, payloadBuf)
	}
	//写入头部
	finalBuf := new(bytes.Buffer)
	err = writeMessageHeader(finalBuf, &msg.header, buf, int32(len(payloadBuf.Bytes())))
	if err != nil {
		return err
	}
	//写入载荷
	finalBuf.Write(payloadBuf.Bytes())
	_, err = writer.Write(finalBuf.Bytes())
	return
}

func (msg *Connect) Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	msg.header = header
	//剩余长度
	remainLen := header.remainLen
	//协议名
	msg.protocolName = getString(reader, &remainLen)
	if msg.protocolName != ProtocolName {
		return NewMessageError(invalidProNameError + ":" + msg.protocolName)
	}
	//协议版本号
	msg.protocolVersion = getUint8(reader, &remainLen)
	if msg.protocolVersion > ProtocolVersion {
		return NewMessageError(fmt.Sprintf(invalidProVersionError+":%d", msg.protocolVersion))
	}
	//标志位，暂不使用
	flags := getUint8(reader, &remainLen)
	msg.enablePayloadGzip = flags&0x80 > 0
	if flags != 0 && flags != 128 {
		return NewMessageError(fmt.Sprintf("connect "+invalidFlagError+":%d", flags))
	}
	//保持连接时间
	msg.keepAliveTime = getUint16(reader, &remainLen)
	//内容
	if msg.enablePayloadGzip {
		msg.Payload = getGzipString(reader, &remainLen)
	} else {
		msg.Payload = getString(reader, &remainLen)
	}

	if remainLen != 0 {
		return NewMessageError(fmt.Sprintf("connect "+msgTooLongError+":%d", remainLen))
	}
	return nil
}

/**
 *	回复连接消息
 */
type ReturnCode uint8

const (
	RetCodeAccepted          = ReturnCode(iota) //连接成功
	RetCodeServerUnavailable                    //服务器开小差
	RetCodeBadLoginInfo                         //登陆信息错误
	RetCodeNotAuthorized                        //未登陆
	RetCodeAlreadyConnected                     //已经连接过了，错误状态，服务器会断开连接
	RetCodeConcurrentLogin                      //并发登陆
	RetCodeBadToken                             //token错误
	RetCodeInvalidUid                           //uid错误

	//每增加一个，下面的errors必须同步增加一个

	retCodeFirstInvalid
)

var ConnectionErrors = []error{
	nil, // Connection Accepted (not an error)
	errors.New("Conn unavailable"),
	errors.New("Bad login info"),
	errors.New("Not authorized"),
	errors.New("Already connected"),
	errors.New("Concurrent login"),
	errors.New("Bad token"),
	errors.New("Invalid uid"),
}

func (rc ReturnCode) IsValid() bool {
	return rc >= RetCodeAccepted && rc < retCodeFirstInvalid
}

type ConnAck struct {
	header FixHeader //固定头部
	//flags		uint8		//预留
	ReturnCode ReturnCode //状态码
}

func (msg *ConnAck) Encode(writer io.Writer, proCommon *ProtocolCommon) (err error) {
	msg.header.MsgType = MsgConnAck

	buf := new(bytes.Buffer)
	//标志位，暂不使用
	buf.WriteByte(byte(0))
	//返回码
	setUint8(uint8(msg.ReturnCode), buf)

	finalBuf := new(bytes.Buffer)
	err = writeMessageHeader(finalBuf, &msg.header, buf, 0)
	if err != nil {
		return err
	}
	_, err = writer.Write(finalBuf.Bytes())
	return
}

func (msg *ConnAck) Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	//报头
	msg.header = header
	//剩余长度
	remainLen := header.remainLen
	//标志位，暂不使用
	flags := getUint8(reader, &remainLen)
	if flags != 0 {
		return NewMessageError(fmt.Sprintf("connack "+invalidFlagError+":%d", flags))
	}
	//返回码
	msg.ReturnCode = ReturnCode(getUint8(reader, &remainLen))
	if !msg.ReturnCode.IsValid() {
		return NewMessageError(fmt.Sprintf(badReturnCodeError+":%d", msg.ReturnCode))
	}

	if remainLen != 0 {
		return NewMessageError(fmt.Sprintf("connack "+msgTooLongError+":%d", remainLen))
	}
	return nil
}

/**
 *	心跳包
 */
type PingReq struct {
	header FixHeader //固定头部
}

func (msg *PingReq) Encode(writer io.Writer, proCommon *ProtocolCommon) (err error) {
	msg.header.MsgType = MsgPingReq
	finalBuf := new(bytes.Buffer)
	writeMessageHeader(finalBuf, &msg.header, nil, 0)
	if err != nil {
		return err
	}
	_, err = writer.Write(finalBuf.Bytes())
	return
}

func (msg *PingReq) Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	//报头
	msg.header = header
	//剩余长度
	remainLen := header.remainLen

	if remainLen != 0 {
		return NewMessageError(fmt.Sprintf("pingreq "+msgTooLongError+":%d", remainLen))
	}
	return nil
}

/**
 *	心跳回应包
 */
type PingResp struct {
	header FixHeader //固定头部
}

func (msg *PingResp) Encode(writer io.Writer, proCommon *ProtocolCommon) (err error) {
	msg.header.MsgType = MsgPingResp
	finalBuf := new(bytes.Buffer)
	err = writeMessageHeader(finalBuf, &msg.header, nil, 0)
	if err != nil {
		return err
	}
	_, err = writer.Write(finalBuf.Bytes())
	return
}

func (msg *PingResp) Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	//报头
	msg.header = header
	//剩余长度
	remainLen := header.remainLen

	if remainLen != 0 {
		return NewMessageError(fmt.Sprintf("pingresp "+msgTooLongError+":%d", remainLen))
	}
	return nil
}

/*
 *	断开连接
 */
type DiscType uint8

const (
	DiscTypeNone    = DiscType(iota) //默认类型，客户端发给服务器的时候使用
	DiscTypeKickout                  //踢出登录，服务器发给客户端，客户端应立即注销
)

type Disconnect struct {
	header FixHeader //固定头部
	Type   DiscType
}

func (msg *Disconnect) Encode(writer io.Writer, proCommon *ProtocolCommon) (err error) {
	msg.header.MsgType = MsgDisconnect
	buf := new(bytes.Buffer)
	//类型
	setUint8(uint8(msg.Type), buf)

	finalBuf := new(bytes.Buffer)
	err = writeMessageHeader(finalBuf, &msg.header, buf, 0)
	if err != nil {
		return err
	}
	_, err = writer.Write(finalBuf.Bytes())
	return
}

func (msg *Disconnect) Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	//报头
	msg.header = header
	//剩余长度
	remainLen := header.remainLen
	//类型
	msg.Type = DiscType(getUint8(reader, &remainLen))

	if remainLen != 0 {
		return NewMessageError(fmt.Sprintf("disconnect "+msgTooLongError+":%d", remainLen))
	}
	return nil
}

/**
 *	发送消息
 */
const (
	RLevelNoReply    = ReplyLevel(iota) //不需要回复
	RLevelReplyLater                    //业务逻辑返回后回复
	//RLevelReplyNow			//立刻回复（业务逻辑之前）

	rLevelFirstInvalid
)

type ReplyLevel uint8

func (rLevel ReplyLevel) IsValid() bool {
	return rLevel < rLevelFirstInvalid
}

func (rLevel ReplyLevel) HasId() bool {
	return rLevel == RLevelReplyLater
}

type SendReq struct {
	header     FixHeader  //固定头部
	ReplyLevel ReplyLevel //回复等级（包含在头部中）
	MessageId  uint16     //消息id
	Type       string     //消息类型
	Payload    string     //消息内容
	HasData    bool       //是否有二进制数据
	Data       []byte     //二进制数据
}

func (msg *SendReq) Encode(writer io.Writer, proCommon *ProtocolCommon) (err error) {
	msg.header.MsgType = MsgSendReq
	//标志位
	var flags, hasData uint8
	if msg.HasData {
		hasData = 1
	} else {
		hasData = 0
	}
	flags = uint8(msg.ReplyLevel<<1) | (hasData << 3)
	msg.header.flags = flags

	buf := new(bytes.Buffer)
	//消息id
	setUint16(msg.MessageId, buf)
	//消息类型
	setString(msg.Type, buf)
	//初始化载荷
	payloadBuf := new(bytes.Buffer)
	if proCommon.EnablePayloadGzip {
		setGzipString(msg.Payload, payloadBuf)
	} else {
		setString(msg.Payload, payloadBuf)
	}
	//二进制数据
	dataBuf := new(bytes.Buffer)
	if msg.HasData {
		setData(msg.Data, dataBuf)
	}
	//写入头部
	finalBuf := new(bytes.Buffer)
	err = writeMessageHeader(finalBuf, &msg.header, buf, int32(len(payloadBuf.Bytes())+len(dataBuf.Bytes())))
	if err != nil {
		return err
	}
	//写入载荷
	finalBuf.Write(payloadBuf.Bytes())
	//写入二进制数据
	if msg.HasData {
		finalBuf.Write(dataBuf.Bytes())
	}
	_, err = writer.Write(finalBuf.Bytes())
	return
}

func (msg *SendReq) Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	msg.header = header
	//回复等级
	msg.ReplyLevel = ReplyLevel(header.flags & 0x06 >> 1)
	//是否有二进制数据
	msg.HasData = (header.flags & 0x08 >> 3) == 1
	//剩余长度
	remainLen := header.remainLen
	//消息id
	msg.MessageId = getUint16(reader, &remainLen)
	//消息类型
	msg.Type = getString(reader, &remainLen)
	//内容
	if proCommon.EnablePayloadGzip {
		msg.Payload = getGzipString(reader, &remainLen)
	} else {
		msg.Payload = getString(reader, &remainLen)
	}
	//二进制数据
	if msg.HasData {
		//内部自动判断是否gzip
		msg.Data = getData(reader, &remainLen)
	} else {
		msg.Data = nil
	}

	if remainLen != 0 {
		return NewMessageError(fmt.Sprintf("sendreq "+msgTooLongError+":%d", remainLen))
	}
	return nil
}

/**
 *	发送消息回执
 */
type SendResp struct {
	header    FixHeader //固定头部
	MessageId uint16    //所回复的消息id
	Payload   string    //消息内容
}

func (msg *SendResp) Encode(writer io.Writer, proCommon *ProtocolCommon) (err error) {
	msg.header.MsgType = MsgSendResp

	buf := new(bytes.Buffer)
	//消息id
	setUint16(msg.MessageId, buf)
	//初始化载荷
	payloadBuf := new(bytes.Buffer)
	if proCommon.EnablePayloadGzip {
		setGzipString(msg.Payload, payloadBuf)
	} else {
		setString(msg.Payload, payloadBuf)
	}
	//写入头部
	finalBuf := new(bytes.Buffer)
	err = writeMessageHeader(finalBuf, &msg.header, buf, int32(len(payloadBuf.Bytes())))
	if err != nil {
		return err
	}
	//写入载荷
	finalBuf.Write(payloadBuf.Bytes())
	_, err = writer.Write(finalBuf.Bytes())
	return
}

func (msg *SendResp) Decode(reader io.Reader, header FixHeader, proCommon *ProtocolCommon) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	//剩余长度
	remainLen := header.remainLen
	//消息id
	msg.MessageId = getUint16(reader, &remainLen)
	//内容
	if proCommon.EnablePayloadGzip {
		msg.Payload = getGzipString(reader, &remainLen)
	} else {
		msg.Payload = getString(reader, &remainLen)
	}

	if remainLen != 0 {
		return NewMessageError(fmt.Sprintf("sendresp "+msgTooLongError+":%d", remainLen))
	}
	return nil
}
