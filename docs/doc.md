# Go-socket Quick Start

## Contents
- [Controller](#controller)
- [Client](#client)
- [Auth](#auth)
- [Error Handling](#error-handling)
- [Log](#log)
- [Build an IM Server](#build-an-im-server)

## Controller
While the go-socket server is listening to the target address and port, it can handle different requests by routing them to different controllers and actions.
We can create controllers inherited from `gosocket.Controller` to handle requests made by the clients. Here is an example of `ChatController` in `chat.go`:

```go
package main

import (
	"fmt"
	gosocket "github.com/yankawayu/go-socket"
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
	fmt.Println(request.Message)
	response.Data = map[string]string{
		"message_id": "1",
	}
	response.Status = gosocket.StatusSuccess
}
```
In this class, the `GetActionParamMap` function is used to declare all the actions that can be handled by the controller.
Then we add this `ChatController` to the router by inserting a new line to `example.go`:
```go
package main

import (
	"github.com/yankawayu/go-socket"
)

func main() {
	appConfig := &gosocket.AppConfig{
		TcpAddr:   "0.0.0.0",
		TcpPort:   8080,
		TlsEnable: false,
	}
	fastLog := gosocket.GetFastLog("app.access", false)
	//Add ChatController to the router
	gosocket.Router("chat", &ChatController{})
	gosocket.Run(appConfig, nil, gosocket.GetLog(false), fastLog)
}
```

It is recommended that the router name `chat` is the same as the prefix of `ChatController`. Now we can access to the `AddMessage` action by using payload type `chat.AddMessage` in a client request.

## Client
Go-socket has a built-in client. It's implemented by socket_client.go and socket_client_conn.go. Let's create a `client.go` that can be used to connect to the server we just created in the last section.
```go
package main

import (
	"fmt"
	gosocket "github.com/yankawayu/go-socket"
)

func main() {
	client := gosocket.NewClient("127.0.0.1", 8080, false, gosocket.GetLog(false))
	err := client.Connect()
	if err != nil {
		panic(err)
	}
	client.GetData("chat.AddMessage", map[string]string{
		"message": "test",
	}, func(err error, data string) {
		fmt.Println(data)
	}, []byte{})
	//Stop the client from exiting before the server responds
	forever := make(chan int)
	_ = <-forever
}
```
The client first connect to the server and then send a request on `chat.AddMessage`. If everything goes well, you will see the server response printed in the console:

```json
{"message_id":"1"}
```

## Auth
In the example above, there is no identification when the client connects to server. In fact, you can create a class that inherited from `AuthUser` to implement identification process.

## Error Handling

## Log

## Build an IM Server