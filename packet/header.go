package packet

import (
	"bytes"
	"fmt"
	"io"
)

type FixHeader struct {
	MsgType   MessageType
	remainLen int32
	flags     uint8
}

func (header *FixHeader) Encode(writer io.Writer) (err error) {
	buf := new(bytes.Buffer)
	err = header.EncodeInto(buf)
	if err != nil {
		return err
	}
	_, err = writer.Write(buf.Bytes())
	return err
}

func (header *FixHeader) EncodeInto(buf *bytes.Buffer) (err error) {
	if !header.MsgType.IsValid() {
		return NewMessageError(fmt.Sprintf("header "+badMsgTypeError+":%d", header.MsgType))
	}
	//消息类型和标志位
	val := (byte(header.MsgType) << 4) | (header.flags & 0x0F)
	buf.WriteByte(val)
	//剩余长度
	encodeLength(header.remainLen, buf)
	return nil
}

func (header *FixHeader) Decode(reader io.Reader) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = GetRecoverError(e)
		}
	}()
	var buf [1]byte
	if _, err = io.ReadFull(reader, buf[:]); err != nil {
		return
	}
	//消息类型
	msgType := MessageType(buf[0] & 0xF0 >> 4)
	//标志位
	flags := buf[0] & 0x0F
	//第四位是保留位，不为0则报错
	if (flags & 0x01) != 0 {
		err = NewMessageError(fmt.Sprintf(invalidFlagError+":%d", flags))
		return
	}
	//剩余长度
	remainingLength, _ := decodeLength(reader)
	//固定头部
	*header = FixHeader{
		MsgType:	msgType,
		remainLen:	remainingLength,
		flags:		flags,
	}
	return
}