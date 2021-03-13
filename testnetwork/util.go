package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type Row struct {
	Description string
	Source      string
	SourceUser  string
	SourceKey   string
	TargetIP    string
	TargetPort  string
	Protocol    string
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
func TransferFile(user string, remote string, port string, key string, src string, dest string) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			PrivateKeyFile(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", remote+port, config)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
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
