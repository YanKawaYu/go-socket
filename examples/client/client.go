package main

import (
	"fmt"
	gosocket "github.com/yankawayu/go-socket"
)

func main() {
	client := gosocket.NewClient("127.0.0.1", 8080, false, gosocket.GetLog(false), &ClientProvider{})
	err := client.Connect()
	if err != nil {
		panic(err)
	}
	client.GetData("chat.AddMessage", map[string]string{
		"message": "This is a message",
	}, func(err error, data string) {
		fmt.Println("Content from server: "+data)
	}, []byte{})
	//Stop the client from exiting before the server responds
	forever := make(chan int)
	_ = <-forever
}
