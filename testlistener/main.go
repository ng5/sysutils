package main

import (
	"flag"
	"fmt"
	"github.com/ng5/sysutils/shared"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"log"
	"strings"
	"time"
)

type tcpServer struct {
	*gnet.EventServer
	pool *goroutine.Pool
}

func (es *tcpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	data := append([]byte{}, frame...)
	_ = es.pool.Submit(func() {
		c.AsyncWrite(data)
		log.Println("Replying TCP packet: " + strings.ReplaceAll(string(data), "\n", ""))
	})
	return
}

type udpServer struct {
	*gnet.EventServer
	pool *goroutine.Pool
}

func (udp *udpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	data := append([]byte{}, frame...)
	_ = udp.pool.Submit(func() {
		c.SendTo(data)
		log.Println("Replying UDP packet: " + strings.ReplaceAll(string(data), "\n", ""))
	})
	return
}

func main() {
	USAGE := "Usage: testlistener -t <TCP> -u <UDP> -m <MULTICAST GROUP:PORT>"
	t := flag.String("t", "12001", "tcp port")
	u := flag.String("u", "12002", "udp port")
	m := flag.String("m", "239.0.0.1:12003", "multicast group & port")
	flag.Parse()
	if len(*t) == 0 || len(*u) == 0 || len(*m) == 0 {
		fmt.Println(USAGE)
		return
	}
	tokens := strings.Split(*m, ":")
	if len(tokens) != 2 {
		fmt.Println(USAGE)
		return
	}
	p1 := goroutine.Default()
	defer p1.Release()
	p2 := goroutine.Default()
	defer p2.Release()

	tcp := "tcp://:" + *t
	udp := "udp://:" + *u

	go gnet.Serve(&tcpServer{pool: p1}, tcp)
	go gnet.Serve(&udpServer{pool: p2}, udp)
	go func() {
		_ = shared.MulticastRead(tokens[0], tokens[1], true)
	}()
	go shared.GenerateTraffic(tokens[0], tokens[1], "test multicast")
	log.Printf("listening TCP: %s\n", *t)
	log.Printf("listening UDP: %s\n", *u)
	log.Printf("listening MULTICAST: %s\n", *m)
	log.Printf("generating MULTICAST: %s\n", *m)
	for {
		time.Sleep(10 * time.Second)
	}
}
