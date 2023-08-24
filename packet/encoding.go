package packet

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
)

func getUint8(r io.Reader, packetRemaining *int32) uint8 {
	if *packetRemaining < 1 {
		panic(dataExceedsPacketError)
	}

	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		panic(err)
	}
	*packetRemaining--

	return b[0]
}

func getUint16(r io.Reader, packetRemaining *int32) uint16 {
	if *packetRemaining < 2 {
		panic(dataExceedsPacketError)
	}

	var b [2]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		panic(err)
	}
	*packetRemaining -= 2

	return uint16(b[0])<<8 | uint16(b[1])
}

func getString(r io.Reader, packetRemaining *int32) string {
	strLen, lenLen := decodeLength(r)
	//Minus the size of the length
	//减去长度所占的字节数
	*packetRemaining -= int32(lenLen)

	if int(*packetRemaining) < int(strLen) {
		panic(dataExceedsPacketError)
	}

	b := make([]byte, strLen)
	if _, err := io.ReadFull(r, b); err != nil {
		panic(err)
	}
	*packetRemaining -= int32(strLen)

	return string(b)
}

func getGzipString(r io.Reader, packetRemaining *int32) string {
	gzipLen, lenLen := decodeLength(r)
	//Minus the size of the length
	//减去长度所占的字节数
	*packetRemaining -= int32(lenLen)

	if int(*packetRemaining) < int(gzipLen) {
		panic(dataExceedsPacketError)
	}

	b := make([]byte, gzipLen)
	if _, err := io.ReadFull(r, b); err != nil {
		panic(err)
	}
	*packetRemaining -= int32(gzipLen)

	result := ""
	//To avoid panic
	//只有长度大于0时才进行解析，否则会出错
	if gzipLen > 0 {
		gzipBuf := bytes.NewBuffer(b)
		reader, err := gzip.NewReader(gzipBuf)
		if err != nil {
			panic(err)
		}
		payloadBytes, err := ioutil.ReadAll(reader)
		if err != nil {
			panic(err)
		}
		result = string(payloadBytes)
	}
	return result
}

func getData(r io.Reader, packetRemaining *int32) []byte {
	dataLen, lenLen := decodeLength(r)
	//减去长度所占的字节数
	*packetRemaining -= int32(lenLen)

	if int(*packetRemaining) < int(dataLen) {
		panic(dataExceedsPacketError)
	}

	b := make([]byte, dataLen)
	if _, err := io.ReadFull(r, b); err != nil {
		panic(err)
	}
	*packetRemaining -= int32(dataLen)
	//Check the first two bytes as magic number
	//检查前两个字节，判断二进制数据类型
	if dataLen > 1 {
		//是否gzip的magic number
		if b[0] == 0x1f && b[1] == 0x8b {
			gzipBuf := bytes.NewBuffer(b)
			reader, err := gzip.NewReader(gzipBuf)
			if err != nil {
				panic(err)
			}
			b, err = ioutil.ReadAll(reader)
			if err != nil {
				panic(err)
			}
		}
	}
	return b
}

func setUint8(val uint8, buf *bytes.Buffer) {
	buf.WriteByte(byte(val))
}

func setUint16(val uint16, buf *bytes.Buffer) {
	buf.WriteByte(byte(val & 0xff00 >> 8))
	buf.WriteByte(byte(val & 0x00ff))
}

func setString(val string, buf *bytes.Buffer) {
	length := int32(len(val))
	encodeLength(length, buf)
	buf.WriteString(val)
}

func setGzipString(val string, buf *bytes.Buffer) {
	//gzip
	var b bytes.Buffer
	gzipWriter := gzip.NewWriter(&b)
	_, err := gzipWriter.Write([]byte(val))
	if err != nil {
		panic(err)
	}
	gzipWriter.Flush()
	gzipWriter.Close()
	//长度
	gzipLen := int32(len(b.Bytes()))
	encodeLength(gzipLen, buf)
	buf.Write(b.Bytes())
}

func setData(val []byte, buf *bytes.Buffer) {
	length := int32(len(val))
	encodeLength(length, buf)
	buf.Write(val)
}

func boolToByte(val bool) byte {
	if val {
		return byte(1)
	}
	return byte(0)
}

// 返回解析出来的长度，以及长度所占的字节数
func decodeLength(r io.Reader) (int32, int) {
	var v int32
	var buf [1]byte
	var shift uint
	for i := 0; i < 4; i++ {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			panic(err)
		}

		b := buf[0]
		v |= int32(b&0x7f) << shift

		if b&0x80 == 0 {
			return v, i + 1
		}
		shift += 7
	}
	panic(badLengthEncodingError)
}

func encodeLength(length int32, buf *bytes.Buffer) {
	if length == 0 {
		buf.WriteByte(0)
		return
	}
	for length > 0 {
		digit := length & 0x7f
		length = length >> 7
		if length > 0 {
			digit = digit | 0x80
		}
		buf.WriteByte(byte(digit))
	}
}
