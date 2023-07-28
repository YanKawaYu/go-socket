package gosocket

import (
	"bytes"
	"encoding/json"
)

func JSONEncode(v interface{}) string {
	return string(JSONByteEncode(v))
}

func JSONDecode(jsonStr string, v interface{}) {
	JSONByteDecode([]byte(jsonStr), v)
}

func JSONEncodeSafe(v interface{}) string {
	return string(JSONByteEncodeSafe(v))
}

func JSONByteEncodeSafe(v interface{}) []byte {
	b := JSONByteEncode(v)
	b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
	b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
	b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	return b
}

func JSONByteEncode(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func JSONByteDecode(jsonByte []byte, v interface{}) {
	err := json.Unmarshal(jsonByte, v)
	if err != nil {
		panic(err)
	}
}
