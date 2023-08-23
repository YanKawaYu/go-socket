package main

type ClientProvider struct{}

func (provider *ClientProvider) GetConnectInfo() string {
	return "{\"username\":\"haha\", \"password\":\"xxxxx\"}"
}
