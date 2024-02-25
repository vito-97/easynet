package main

import (
	"easynet"
	"fmt"
	"os"
	"os/signal"
	"time"
)

const (
	HelloType = iota + 1
	PingType
	DebugType
)

func Debug(req easynet.IRequest) {
	fmt.Println("debug打印", string(req.GetData()))
}

func main() {
	r := easynet.DefaultClient(func(c *easynet.ClientOption) {
		c.OnConnStart = append(c.OnConnStart, func(connection easynet.IConnection) {
			connection.SendMsg(HelloType, []byte("hello"))
			go func() {
				for {
					connection.SendMsg(PingType, []byte("hello server"))
					time.Sleep(time.Second)
				}
			}()
		})
	})

	r.Use(func(req easynet.IRequest) {
		fmt.Println("global middleware")
	})

	group := r.Group("group", func(req easynet.IRequest) {
		fmt.Println("group middleware")
	})

	{
		group.Add(HelloType, func(req easynet.IRequest) {
			fmt.Println("hello world")
		})
	}

	r.Add(PingType, func(req easynet.IRequest) {
		fmt.Println("ping,msg=", string(req.GetData()))
	})

	r.Add(DebugType, Debug)

	r.Run()

	// close
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, os.Kill)
	sig := <-sc
	fmt.Println("====client exit====", sig)
	r.Stop()
	time.Sleep(time.Second * 2)
}