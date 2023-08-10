# GOSOC
GOSOC is similar to [MQTT](https://mqtt.org/). It operates by exchanging a series of control packets in a defined way. This document will describe the format of these packets.

## Limit
The maximum length of a control packet is 256MB.
```go
const (
	// Maximum payload size in bytes (256MiB - 1B).
	MaxPayloadSize = (1 << (4 * 7)) - 1
)
```

## Data representations
The data representations of `GOSOC` are exactly the same as MQTT. Please refer the [documents](https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901006) for more information. 

## Components
A control packet consists of up to four parts, including fixed header, variable header, payload, binary data. Among them variable header, payload and binary data can be empty according to different kinds of packets.

| Packet Structure |
|------------------|
| Fixed Header     |
| Variable Header  |
| Payload          |
| Binary Data      |

The length of a fixed header is fixed as 6 bytes. The structure of a fixed header is shown in the following:

```go
type FixHeader struct {
	MsgType   uint8		//Message type
	remainLen int32		//Remaining length of the packet
	flags     uint8		//Flags, reserved
}
```

The length of a variable header varies from type to type. But once we know the message type of the packet, we will be able to know the length of a variable header according to the rest of the documents.

## Message Types
The following are all the message types that supported.
```go
const (
	MsgConnect = 1  //Connect message
	MsgConnAck      //Response to Connect message
	MsgPingReq      //Ping-pong message
	MsgPingResp     //Response to Ping-pong message
	MsgDisconnect   //Disconnect message
	MsgSendReq      //Request message
	MsgSendResp     //Response to Request message
)
```

## Messages
### Connect
For connect message, it consists of fixed header, variable header and payload. The variable header includes ProtocolName, ProtocolVersion, Flags and KeepAliveTime.
```go
type Connect struct {
	header          FixHeader   //Fixed header
	ProtocolName	string      //Protocol name, GOSOC
	ProtocolVersion	uint8       //Protocol version, 1
	Flags           uint8       //The 7th bit is used to mark whether to enable gzip
	KeepAliveTime	uint16      //Ping-pong message interval
	Payload         string      //JSON
}
```
The payload can be a JSON string including login information and token. For example:
```json
{
    "auth_key": "U2FsdGVkX1+MlB0YKIvPd17NI6XSRCQ0daGphqp/vkqZoCEQpBQHr+qPYpO3e67a/88sGOFi5A0Ougd9SUPiKVBZSx2F5o7EtE2VhuSAuI0=",
    "token": "42728ff2118430bdff5f9a189e0034ec"
}
```

### ConnAck
For connack message, it consists of fixed header and variable header. The Flags and ReturnCode are belong to the variable header.
```go
type ConnAck struct {
	header		FixHeader	//Fixed header
	Flags		uint8		//Reserved
	ReturnCode	uint8		//Status code
}
```
The ReturnCode can be the following types:
```go
const (
	RetCodeAccepted = ReturnCode(iota)      //Connect successfully
	RetCodeServerUnavailable                //Server is currently unavailable
	RetCodeBadLoginInfo                     //There are some problems with login information
	RetCodeNotAuthorized                    //The connection hasn't been authorized
	RetCodeAlreadyConnected                 //This happens when the server receives duplicated Connect message
	RetCodeConcurrentLogin                  //This happens when the server receives two concurrent logins from same user 
	RetCodeBadToken                         //There are some problems with token
	RetCodeInvalidUid                       //Uid is invalid
)
```

### PingReq
For pingreq message, it's designed to be as small as possible. So it only has a fixed header.
```go
type PingReq struct {
	header		FixHeader //Fixed header
}
```

### PingResp
For pingresp message, it's similar to PingReq message.
```go
type PingResp struct {
	header		FixHeader //Fixed header
}
```

### Disconnect
For disconnect message, the variable header only consists of Type.
```go
type Disconnect struct {
	header		FixHeader //Fixed header
	Type		uint8
}
```
The Type can be the following types:
```go
const (
	DiscTypeNone = DiscType(iota) //the default one, sent by the client to close connection
	DiscTypeKickout //server use this one to ask the client to disconnect immediately
)
```

### SendReq
For sendreq message, the variable header consists MessageId and Type. The Type is used as a route similar to http url. 

For example, the Type `chat.AddMessage` refers that the request should be handled by the `AddMessage` action under ChatController. Read the [Route](../docs/doc.md) section in Go-socket Quick Start for more information.

You need to pay attention that ReplyLevel is not belong to the variable header. It's decided by the 2nd and 3rd bit of flags in the fixed header.
```go
type SendReq struct {
	header		FixHeader 	//Fixed header
	ReplyLevel	ReplyLevel 	//Reply level(Belongs to fixed header)
	MessageId	uint16 		//Message id
	Type		string		//Request route
	Payload		string		//JSON
	Data		[]byte		//binary data
}
```
The ReplyLevel can be the following types:
```go
const (
	RLevelNoReply = ReplyLevel(iota) //message that don't need to reply
	RLevelReplyLater //message that should be replied after the action
)
```
Antother thing you need know is that the 4th bit in the flags of fixed header is used to mark whether there is binary data after the payload. The binary data here works similar as attachment in http protocol.
You can use the `data` field to perform an upload action here.

### SendResp
For sendresp message, the MessageId is the only field in the variable header. It is required that the sendresp message and its corresponding sendreq message share the same message id, so that the server will pair the sendresp message with the right sendreq message.
```go
type SendResp struct {
	header		FixHeader 	//Fixed header
	MessageId	uint16 		//Message id to respond
	Payload		string		//JSON
}
```