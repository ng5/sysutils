package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ng5/sysutils/shared"
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

var hostName string

func IsLocal(src string) bool {
	if strings.ToLower(src) == "localhost" || src == "127.0.0.1" || src == hostName {
		return true
	}
	return false
}
func printRow(row *shared.Row, err error) {
	fmt.Printf("%-12s %-20s %-20s %-20s %-12s %v\n", row.Description, hostName, row.SourceUser+"@"+row.Source, row.TargetIP+":"+row.TargetPort, row.Protocol, err)
}
func main() {
	hostName, _ = os.Hostname()
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath, err := filepath.Abs(ex)
	exBaseName := filepath.Base(ex)
	csvFile := flag.String("file", "", "csv file containing rules")
	remote := flag.Bool("remote", false, "run test on remote machines")
	overwrite := flag.Bool("overwrite", true, "overwrite rules file and itself on remote machines")
	flag.Parse()
	if len(*csvFile) == 0 {
		fmt.Println("Usage: testnetwork --file <csv file>")
		return
	}
	U, err := user.Current()
	lines, err := shared.ReadCsv(*csvFile)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	count := 0
	rows := make([]shared.Row, 0)
	m := map[string]shared.Row{}
	for _, line := range lines {
		count++
		if count == 1 {
			continue
		}
		row := shared.Row{
			Description: line[0],
			Source:      line[1],
			SourceUser:  line[2],
			SourceKey:   line[3],
			TargetIP:    line[4],
			TargetPort:  line[5],
			Protocol:    line[6],
		}
		if len(row.SourceUser) == 0 {
			row.SourceUser = U.Username
		}
		if len(row.SourceKey) == 0 {
			row.SourceKey = path.Join(U.HomeDir, ".ssh", "id_rsa")
		}
		rows = append(rows, row)
		previous, ok := m[row.Source]
		if ok {
			// Don't replace user added entry
			if previous.SourceUser != U.Username {
				continue
			}
		}
		m[row.Source] = row

	}
	if *remote == true {
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
			row := m[k]
			if IsLocal(row.Source) {
				continue
			}
			if *overwrite == true {
				err := shared.TransferFile(row.SourceUser, row.Source, ":22", row.SourceKey, csvFileBase, "/tmp/"+csvFileBase)
				if err != nil {
					printRow(&row, err)
					continue
				}
				err = shared.TransferFile(row.SourceUser, row.Source, ":22", row.SourceKey, exPath, "/tmp/"+exBaseName)
				if err != nil {
					printRow(&row, err)
					continue
				}
			}
			err = shared.RemoteExecSSH(row.SourceUser, row.Source, ":22", row.SourceKey, "chmod +x /tmp/"+exBaseName)
			if err != nil {
				printRow(&row, err)
				continue
			}
			err = shared.RemoteExecSSH(row.SourceUser, row.Source, ":22", row.SourceKey, "/tmp/"+exBaseName+" --file "+"/tmp/"+csvFileBase)
			if err != nil {
				printRow(&row, err)
				continue
			}
		}
	} else {
		fmt.Printf("%-12s %-20s %-20s %-20s %-12s %-12s\n", "Description", "HostName", "Source", "Target", "Protocol", "Status")
		fmt.Printf("-------------------------------------------------------------------------------------------------------\n")
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
