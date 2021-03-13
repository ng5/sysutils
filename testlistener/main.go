package main

import (
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"log"
	"strconv"
	"strings"
	"time"
)

type echoServer struct {
	*gnet.EventServer
	pool *goroutine.Pool
}

func (es *echoServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	data := append([]byte{}, frame...)
	_ = es.pool.Submit(func() {
		log.Println("TCP packet: " + strings.ReplaceAll(string(data), "\n", ""))
		c.AsyncWrite(data)
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
		log.Println("UDP packet: " + strings.ReplaceAll(string(data), "\n", ""))
		c.SendTo(data)
	})
	return
}

type multicastServer struct {
	*gnet.EventServer
	pool *goroutine.Pool
}

func (multicast *multicastServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	data := append([]byte{}, frame...)
	_ = multicast.pool.Submit(func() {
		log.Println("MULTICAST packet: " + strings.ReplaceAll(string(data), "\n", ""))
	})
	return
}
func main() {
	p1 := goroutine.Default()
	defer p1.Release()
	p2 := goroutine.Default()
	defer p2.Release()
	p3 := goroutine.Default()
	defer p3.Release()

	tcp := 12001
	udp := 12002
	multicast := 12003
	go gnet.Serve(&echoServer{pool: p1}, "tcp://:"+strconv.Itoa(tcp))
	go gnet.Serve(&udpServer{pool: p2}, "udp://:"+strconv.Itoa(udp))
	go gnet.Serve(&multicastServer{pool: p3}, "udp://224.0.0.1:"+strconv.Itoa(multicast))
	log.Printf("listening TCP: %d\n", tcp)
	log.Printf("listening UDP: %d\n", udp)
	log.Printf("listening MULTICAST: %d\n", multicast)
	for {
		time.Sleep(10 * time.Second)
	}
}
