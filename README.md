# go-socket
Go-socket is an open-source, high-performance socket framework for building backend services in Golang.

The protocol of Go-socket is called `GOSOC`, which is similar to [MQTT](https://mqtt.org/). Since MQTT is designed for the Internet of Things(loT), it's extremely efficient. For more information, please read [GOSOC](docs/gosoc.md).
## Getting started

### Getting Go-socket

With [Go module](https://github.com/golang/go/wiki/Modules) support, simply add the following import

```
import "github.com/YanKawaYu/go-socket"
```

to your code, and then `go [build|run|test]` will automatically fetch the necessary dependencies.

Otherwise, run the following Go command to install the `go-socket` package:

```sh
$ go get -u github.com/YanKawaYu/go-socket
```

If you hadn't created a module, make sure you run this command first

```sh
go mod init Example
```

### Running Go-socket

After you import go-socket package, you can start with a simplest example like the following `example.go`:

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
	// listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	gosocket.Run(appConfig, nil, gosocket.GetLog(false), fastLog)
}
```
Make sure you create a `runtime` directory for logging and then use the Go command to run the demo:

```
# create a directory for logging
$ mkdir -m 777 runtime
# run example.go and access the server with telnet
$ go run example.go
$ telnet 127.0.0.1 8080
```

### Go versions
Since we use `go.uber.org/zap` as the log component, it only supports the two most recent minor versions of Go. Therefore, the requirement of the Go version for this framework is the same.

### Learn more examples

Learn and practice more examples, please read the [Go-socket Quick Start](docs/doc.md) which includes API examples

## Contributing

For now, I'm the only one that maintaining Go-socket. Any pull requests, suggestions or issues are appreciated!

## License

Go-socket is under the MIT license. See the [LICENSE](/LICENSE) file for details.