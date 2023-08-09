package packet

import (
	"fmt"
	"io"
)

type ProtocolCommon struct {
	ProName           string
	ProVersion        uint8
	KeepAliveTime     uint16
	EnablePayloadGzip bool
}

// MessageManager is the class used to decode and encode messages
type MessageManager struct {
	ProCommon ProtocolCommon
}

// newMessage create a new message
// 创建一条新消息
func newMessage(msgType MessageType) (msg IMessage, err error) {
	switch msgType {
	case MsgConnect:
		msg = new(Connect)
	case MsgConnAck:
		msg = new(ConnAck)
	case MsgPingReq:
		msg = new(PingReq)
	case MsgPingResp:
		msg = new(PingResp)
	case MsgDisconnect:
		msg = new(Disconnect)
	case MsgSendReq:
		msg = new(SendReq)
	case MsgSendResp:
		msg = new(SendResp)
	default:
		return nil, NewMessageError(fmt.Sprintf(badMsgTypeError+":%d", msgType))
	}
	return
}

// DecodeMessage reads a new message from the io
// 读取一条新消息
func (manager *MessageManager) DecodeMessage(reader io.Reader) (msg IMessage, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	var header FixHeader
	err = header.Decode(reader)
	if err != nil {
		return
	}
	msg, err = newMessage(header.MsgType)
	if err != nil {
		return
	}
	switch message := msg.(type) {
	//If the message is a Connect, save the common params
	//如果是Connect包，将包中数值赋值给协议的公共参数，另外不需要传入公共参数
	case *Connect:
		err = msg.Decode(reader, header, nil)
		manager.ProCommon.ProName = message.protocolName
		manager.ProCommon.ProVersion = message.protocolVersion
		manager.ProCommon.KeepAliveTime = message.keepAliveTime
		manager.ProCommon.EnablePayloadGzip = message.enablePayloadGzip
	default:
		//其他的包直接传入公共参数
		err = msg.Decode(reader, header, &manager.ProCommon)
	}
	return msg, err
}

// EncodeMessage writes a new message into the io
// 写入一条消息
func (manager *MessageManager) EncodeMessage(writer io.Writer, msg IMessage) error {
	var err error = nil
	switch message := msg.(type) {
	//If the message is a Connect, assign the common params
	//如果是Connect包，将协议的公共参数赋值到包中数值，另外不需要传入公共参数
	case *Connect:
		message.protocolName = manager.ProCommon.ProName
		message.protocolVersion = manager.ProCommon.ProVersion
		message.keepAliveTime = manager.ProCommon.KeepAliveTime
		message.enablePayloadGzip = manager.ProCommon.EnablePayloadGzip
		err = msg.Encode(writer, nil)
	default:
		//其他的包直接传入公共参数
		err = msg.Encode(writer, &manager.ProCommon)
	}
	return err
}
