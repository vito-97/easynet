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
	r := easynet.DefaultServer(easynet.ServerWithOnConnStart(func(connection easynet.IConnection) {
		go func() {
			for !connection.IsStopped() {
				connection.SendMsg(PingType, []byte("hello client"))
				time.Sleep(time.Second)
			}
		}()
	}))

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
		log.Println("ping,msg=", string(req.Data()))
	})

	r.Run()
}
