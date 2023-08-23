package main

import (
	gosocket "github.com/yankawayu/go-socket"
	"github.com/yankawayu/go-socket/packet"
)

type TestUser struct {
	gosocket.AuthUser
}

func (user *TestUser) Auth(payload string, ip string) (uid int64, code packet.ReturnCode) {
	loginInfo := map[string]string{
		"username": "xxx",
		"password": "xxx",
	}
	gosocket.JSONDecode(payload, &loginInfo)
	uid = auth(loginInfo["username"], loginInfo["password"])
	return uid, packet.RetCodeAccepted
}

func auth(string, string) int64 {
	return 1
}
