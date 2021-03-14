package shared

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/ipv4"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"
)

const TimeoutSeconds = 2

type Row struct {
	Description string
	Source      string
	SourceUser  string
	SourceKey   string
	TargetIP    string
	TargetPort  string
	Protocol    string
	TimeOut     int
}

func PrivateKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}

	return ssh.PublicKeys(key)
}
func ReadCsv(filename string) ([][]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return [][]string{}, err
	}
	defer f.Close()
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return [][]string{}, err
	}
	return lines, nil
}
func TransferFile(user string, remote string, port string, key string, src string, dest string) error {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			PrivateKeyFile(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", remote+port, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	dstFile, err := client.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %s %d bytes copied\n", remote, srcFile.Name(), bytes)
	return nil
}
func RemoteExecSSH(user string, remote string, port string, key string, command string) error {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			PrivateKeyFile(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", remote+port, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	err = session.Run(command)
	str := b.String()
	if len(str) > 0 {
		fmt.Println(str)
	}
	return nil
}

func MulticastRead(group, port string, timeout time.Duration, continuous bool) error {
	p, err1 := strconv.Atoi(port)
	if err1 != nil {
		log.Fatal(err1)
	}
	a := net.ParseIP("0.0.0.0")
	g := net.ParseIP(group)
	if g == nil {
		log.Fatal(fmt.Errorf("bad group: '%s'", group))
	}
	c, err3 := MulticastOpen(a, p)
	if err3 != nil {
		log.Fatal(err3)
	}
	defer c.Close()
	if err := c.JoinGroup(nil, &net.UDPAddr{IP: g}); err != nil {
		log.Fatal(err)
	}
	if err := c.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		log.Fatal(err)
	}
	return MulticastLoop(c, timeout, continuous)
}

func MulticastOpen(bindAddr net.IP, port int) (*ipv4.PacketConn, error) {
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
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	p := ipv4.NewPacketConn(c)
	return p, nil
}

func MulticastLoop(c *ipv4.PacketConn, timeout time.Duration, continuous bool) error {
	buf := make([]byte, 10000)
	for {
		c.SetReadDeadline(time.Now().Add(timeout))
		n, cm, _, err1 := c.ReadFrom(buf)
		if err1 != nil {
			if !continuous {
				return err1
			} else {
				log.Printf("MULTICAST: readfrom: error %v", err1)
				continue
			}
		} else {
			if !continuous {
				return nil
			}
		}
		var name string
		ifi, err2 := net.InterfaceByIndex(cm.IfIndex)
		if err2 != nil {
			if !continuous {
				return err1
			} else {
				log.Printf("readLoop: unable to solve ifIndex=%d: error: %v", cm.IfIndex, err2)
			}
		}

		if ifi == nil {
			name = "ifname?"
		} else {
			name = ifi.Name
		}
		if continuous {
			log.Printf("MULTICAST packet: recv %d bytes from %s to %s on %s", n, cm.Src, cm.Dst, name)
		}
	}
}
func GenerateTraffic(group string, port string, data string) {
	conn, err := net.Dial("udp", group+":"+port)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		_, err = conn.Write([]byte(data))
		if err != nil {
			return
		} else {
			time.Sleep(time.Second)
		}
	}
}
