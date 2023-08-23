package main

import (
	"fmt"
	gosocket "github.com/yankawayu/go-socket"
	"strconv"
)

type ChatController struct {
	gosocket.Controller
}

type AddMessageReqBody struct {
	Message string `json:"message"`
}

func (controller *ChatController) GetActionParamMap() map[string]interface{} {
	return map[string]interface{}{
		"AddMessage": &AddMessageReqBody{},
	}
}

func (controller *ChatController) AddMessage(request *AddMessageReqBody, response *gosocket.ResponseBody) {
	fmt.Println("user " + strconv.FormatInt(controller.User.GetUid(), 10) + " message received: " + request.Message)
	response.Data = map[string]string{
		"message_id": "1",
	}
	response.Status = gosocket.StatusSuccess
}
