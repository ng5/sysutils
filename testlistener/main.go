package main

import (
	"flag"
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pool/goroutine"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type tcpServer struct {
	*gnet.EventServer
	pool *goroutine.Pool
}

func (es *tcpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
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

func mcastRead(group, port string) {
	p, err1 := strconv.Atoi(port)
	if err1 != nil {
		log.Fatal(err1)
	}
	a := net.ParseIP("0.0.0.0")
	g := net.ParseIP(group)
	if g == nil {
		log.Fatal(fmt.Errorf("bad group: '%s'", group))
	}
	c, err3 := mcastOpen(a, p)
	if err3 != nil {
		log.Fatal(err3)
	}
	if err := c.JoinGroup(nil, &net.UDPAddr{IP: g}); err != nil {
		log.Fatal(err)
	}
	if err := c.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		log.Fatal(err)
	}
	readLoop(c)
	c.Close()
}

func mcastOpen(bindAddr net.IP, port int) (*ipv4.PacketConn, error) {
	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_UDP)
	if err != nil {
		log.Fatal(err)
	}
	if err := syscall.SetsockoptInt(s, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		log.Fatal(err)
	}
	lsa := syscall.SockaddrInet4{Port: port}
	copy(lsa.Addr[:], bindAddr.To4())
	if err := syscall.Bind(s, &lsa); err != nil {
		syscall.Close(s)
		log.Fatal(err)
	}
	f := os.NewFile(uintptr(s), "")
	c, err := net.FilePacketConn(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	p := ipv4.NewPacketConn(c)
	return p, nil
}

func readLoop(c *ipv4.PacketConn) {
	buf := make([]byte, 10000)
	for {
		n, cm, _, err1 := c.ReadFrom(buf)
		if err1 != nil {
			log.Printf("MULTICAST: readfrom: error %v", err1)
			break
		}
		var name string
		ifi, err2 := net.InterfaceByIndex(cm.IfIndex)
		if err2 != nil {
			log.Printf("readLoop: unable to solve ifIndex=%d: error: %v", cm.IfIndex, err2)
		}

		if ifi == nil {
			name = "ifname?"
		} else {
			name = ifi.Name
		}
		log.Printf("MULTICAST packet: recv %d bytes from %s to %s on %s", n, cm.Src, cm.Dst, name)
	}
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
		mcastRead(tokens[0], tokens[1])
	}()
	log.Printf("listening TCP: %s\n", *t)
	log.Printf("listening UDP: %s\n", *u)
	log.Printf("listening MULTICAST: %s\n", *m)
	for {
		time.Sleep(10 * time.Second)
	}
}
