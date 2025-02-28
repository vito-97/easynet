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
	DebugType
)

func Debug(req easynet.IRequest) {
	fmt.Println("debug打印", string(req.Data()))
}

func main() {
	r := easynet.DefaultClient(func(c *easynet.ClientOption) {
		c.OnConnStart = append(c.OnConnStart, func(connection easynet.IConnection) {
			connection.SendMsg(HelloType, []byte("hello"))
			helloData := []byte("hello everyone~！")
			msg := easynet.NewMessageWithType(HelloType, helloData)
			pack := easynet.NewDataPack()

			bytes, _ := pack.Pack(msg)
			b := append(bytes, bytes...)
			connection.Send(b)

			go func() {
				for !connection.IsStopped() {
					connection.SendMsg(PingType, []byte("hello server"))
					time.Sleep(time.Second)
				}
			}()
		})
	})

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
