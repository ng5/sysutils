package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func IsLocal(src string) bool {
	hostName, _ := os.Hostname()
	if strings.ToLower(src) == "localhost" || src == "127.0.0.1" || src == hostName {
		return true
	}
	return false
}

func main() {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath, err := filepath.Abs(ex)
	exBaseName := filepath.Base(ex)
	csvFile := flag.String("file", "", "csv file")
	replicate := flag.Bool("replicate", false, "run test on remote machines")
	overwrite := flag.Bool("overwrite", true, "overwrite rules file and itself on remote machines")
	flag.Parse()
	if len(*csvFile) == 0 {
		fmt.Println("Usage: testnetwork --file <csv file>")
		return
	}
	U, err := user.Current()
	lines, err := ReadCsv(*csvFile)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	count := 0
	rows := make([]Row, 0)
	m := map[string]Row{}
	for _, line := range lines {
		count++
		if count == 1 {
			continue
		}
		row := Row{
			Description: line[0],
			Source:      line[1],
			SourceUser:  line[2],
			SourceKey:   line[3],
			TargetIP:    line[4],
			TargetPort:  line[5],
			Protocol:    line[6],
		}
		if len(row.SourceUser) == 0 {
			row.SourceUser = U.Name
		}
		if len(row.SourceKey) == 0 {
			row.SourceKey = path.Join(U.HomeDir, ".ssh", "id_rsa")
		}
		rows = append(rows, row)
		m[row.Source] = row
	}

	if *replicate == true {
		if runtime.GOOS != "linux" {
			fmt.Println("replication is only allowed from linux machine")
			return
		}
		csvFileBase := filepath.Base(*csvFile)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := m[k]
			if IsLocal(v.Source) {
				continue
			}
			if *overwrite == true {
				TransferFile(v.SourceUser, v.Source, ":22", v.SourceKey, csvFileBase, "/tmp/"+csvFileBase)
				TransferFile(v.SourceUser, v.Source, ":22", v.SourceKey, exPath, "/tmp/"+exBaseName)
			}
			_ = RemoteExecSSH(v.SourceUser, v.Source, ":22", v.SourceKey, "chmod +x /tmp/"+exBaseName)
			_ = RemoteExecSSH(v.SourceUser, v.Source, ":22", v.SourceKey, "/tmp/"+exBaseName+" --file "+"/tmp/"+csvFileBase)
		}
	} else {
		fmt.Printf("%-12s %-20s %-20s %-20s %-12s %-12s\n", "Description", "HostName", "Source", "Target", "Protocol", "Status")
		fmt.Printf("-------------------------------------------------------------------------------------------------------\n")
		hostName, _ := os.Hostname()
		d := net.Dialer{Timeout: 1 * time.Second}
		for _, row := range rows {
			status := ""
			if !IsLocal(row.Source) {
				continue
			}
			conn, err := d.Dial(strings.ToLower(row.Protocol), row.TargetIP+":"+row.TargetPort)
			if err != nil {
				status = "FAILED: " + err.Error()
			} else {
				_, err = conn.Write([]byte("test\n"))
				if err != nil {
					status = "FAILED " + err.Error()
				} else {
					if row.Protocol == strings.ToLower("tcp") {
						_, _, err = bufio.NewReader(conn).ReadLine()
						if err != nil {
							status = "FAILED " + err.Error()
						} else {
							status = "OK"
						}
					} else {
						status = "OK"
					}
				}
			}
			if conn != nil {
				_ = conn.Close()
			}
			fmt.Printf("%-12s %-20s %-20s %-20s %-12s %-12s\n", row.Description, hostName, row.Source, row.TargetIP+":"+row.TargetPort, row.Protocol, status)
		}
	}
}
