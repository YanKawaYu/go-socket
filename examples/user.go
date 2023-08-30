package main

import (
	"fmt"
	gosocket "github.com/yankawayu/go-socket"
	"github.com/yankawayu/go-socket/packet"
	"strconv"
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
	fmt.Println("user " + strconv.FormatInt(uid, 10) + " connected")
	return uid, packet.RetCodeAccepted
}

func auth(string, string) int64 {
	return 1
}
