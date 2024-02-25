package main

import (
	"easynet"
	"fmt"
	"time"
)

const (
	HelloType = iota + 1
	PingType
)

func main() {
	r := easynet.Default(func(s *easynet.ServerOption) {
		s.OnConnStart = append(s.OnConnStart, func(connection easynet.IConnection) {
			go func() {
				for {
					connection.SendMsg(PingType, []byte("hello client"))
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

	r.Run()
}