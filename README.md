## EasyNet

EasyNet是一个基于Golang的轻量级并发服务器框架

**版本**
Golang 1.20+

下载EasyNet包

```bash
$ go get github.com/vito-97/easynet
```

> note: Golang Version 1.20+

#### EasyNet服务器

```go
package main

import (
	"github.com/vito-97/easynet"
	"log"
	"time"
)

const (
	PingType = iota + 1
	HelloType
)

func main() {
	r := easynet.Default()

	r.Use(func(req easynet.IRequest) {
		log.Println("global middleware")
	})

	r.SetOnConnStart(func(connection easynet.IConnection) {
		go func() {
			for !connection.IsStopped() {
				connection.SendMsg(PingType, []byte("ping"))
				time.Sleep(1 * time.Second)
			}
		}()
	})

	group := r.Group("group", func(req easynet.IRequest) {
		log.Println("group middleware")
	})

	{
		group.Add(HelloType, func(req easynet.IRequest) {
			log.Println("hello world")
		})
	}

	r.Add(PingType, func(req easynet.IRequest) {
		log.Println("ping,msg=", string(req.GetData()))
	})

	r.Run()
}


```

#### 启动服务器

```bash
$ go run server.go 
```

```bash
[EasyNet-debug] config file /www/demo/config/easynet.json is not exist!
 You can set config file by setting the environment variable EASYNET_CONFIG_FILE, like export EASYNET_CONFIG_FILE = xxx/xxx/easynet.conf
[EasyNet-debug] route id 2 --> main.main.func3 (5 handlers)
[EasyNet-debug] route id 1 --> main.main.func4 (4 handlers)
[EasyNet-debug] worker is start

```

#### EasyNet客户端

```go
package main

import (
	"fmt"
	"github.com/vito-97/easynet"
	"log"
	"os"
	"os/signal"
	"time"
)

const (
	PingType = iota + 1
	HelloType
)

func main() {
	r := easynet.DefaultClient()

	r.Use(func(req easynet.IRequest) {
		log.Println("global middleware")
	})

	group := r.Group("group", func(req easynet.IRequest) {
		log.Println("group middleware")
	})

	{
		group.Add(HelloType, func(req easynet.IRequest) {
			log.Println("hello world")
		})
	}

	r.Add(PingType, func(req easynet.IRequest) {
		log.Println("ping,msg=", string(req.GetData()))
	})

	r.Run()

	// close
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, os.Kill)
	sig := <-sc
	fmt.Println("====client exit====", sig)
	r.Stop()
	time.Sleep(time.Second * 2)
}

```

#### 启动客户端

```bash
$ go run client.go 
```

```bash
[EasyNet-debug] config file /www/demo/config/easynet.json is not exist!
 You can set config file by setting the environment variable EASYNET_CONFIG_FILE, like export EASYNET_CONFIG_FILE = xxx/xxx/easynet.conf
[EasyNet-debug] route id 2 --> main.main.func3 (5 handlers)
[EasyNet-debug] route id 1 --> main.main.func4 (4 handlers)
[EasyNet-debug] worker is start

```

控制台持续打印输出

```bash
2024/03/09 16:33:43 global middleware
2024/03/09 16:33:43 ping,msg= ping
[EasyNet] 2024-03-09 16:33:43 |  356.5µs |  127.0.0.1:8899 | type 1
ping
2024/03/09 16:33:44 global middleware
2024/03/09 16:33:44 ping,msg= ping
[EasyNet] 2024-03-09 16:33:44 |    135.8µs |  127.0.0.1:8899 | type 1
ping
2024/03/09 16:33:45 global middleware
2024/03/09 16:33:45 ping,msg= ping
[EasyNet] 2024-03-09 16:33:45 |    127.6µs |  127.0.0.1:8899 | type 1
ping
```

### EasyNet配置文件

```json
{
  "Name": "EasyNet Server Demo",
  "ClientName": "EasyNet Client Demo",
  "Host": "0.0.0.0",
  "TCPPort": 8899,
  "MaxConn": 12000,
  "MaxPacketSize": 4096,
  "WorkerPoolSize": 10,
  "MaxWorkerTaskLen": 1024,
  "WorkerMode": "",
  "MaxMsgChanLen": 1024,
  "IOReadBuffSize": 1024
}
```

`Name`:服务器应用名称

`ClientName`:客户端应用名称

`Host`:服务器IP

`TcpPort`:服务器监听端口

`MaxConn`:允许的客户端连接最大数量

`MaxPacketSize`:最大的包大小

`WorkerPoolSize`:工作任务池最大工作Goroutine数量

`MaxWorkerTaskLen`:工作任务池的单个worker的任务的长度

`WorkerMode`:工作任务池的模式，hash取余方式 bind每个连接绑定一个worker

`MaxMsgChanLen`:连接上来的客户端用队列发送时，创建队列的容量大小

`IOReadBuffSize`:每次读取客户端数据的最大长度