# go-socket
Go-socket is an open-source, high-performance socket framework for building backend services in Golang.

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
	gosocket.Run(appConfig, nil, gosocket.GetLog(false), fastLog) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
```
And use the Go command to run the demo:

```
# run example.go and access the server with telnet
$ go run example.go
$ telnet 127.0.0.1 8080
```

### Learn more examples