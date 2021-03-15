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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
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
func ReadRows(file string, filter *string) (map[string]shared.Row, []shared.Row) {
	U, err := user.Current()
	lines, err := shared.ReadCsv(file)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, nil
	}
	count := 0
	rows := make([]shared.Row, 0)
	m := map[string]shared.Row{}
	for _, line := range lines {
		count++
		if count == 1 {
			continue
		}
		timeout, e := strconv.ParseInt(line[7], 10, 64)
		if e != nil {
			timeout = shared.TimeoutSeconds
		}
		row := shared.Row{
			Description: line[0],
			Source:      line[1],
			SourceUser:  line[2],
			SourceKey:   line[3],
			TargetIP:    line[4],
			TargetPort:  line[5],
			Protocol:    line[6],
			TimeOut:     int(timeout),
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
	applyFilter := false
	if filter != nil && len(strings.TrimSpace(*filter)) > 0 {
		applyFilter = true
	}
	if !applyFilter {
		return m, rows
	}
	regEx := regexp.MustCompile(strings.TrimSpace(*filter))
	filteredRows := make([]shared.Row, 0)
	for _, row := range rows {
		if applyFilter && !regEx.MatchString(row.Description) {
			continue
		}
		filteredRows = append(filteredRows, row)
	}
	return m, filteredRows
}
func GetMaxRoutines(m map[string]shared.Row) int {
	maxRoutines := 0
	for _, v := range m {
		if IsLocal(v.Source) {
			continue
		}
		maxRoutines++
	}
	return maxRoutines
}
func RemoteExec(overwrite bool, row *shared.Row, csvFileBase string, ex string, wg *sync.WaitGroup, filter *string) {
	defer wg.Done()
	exPath, err := filepath.Abs(ex)
	exBaseName := filepath.Base(ex)
	if overwrite == true {
		err := shared.TransferFile(row.SourceUser, row.Source, ":22", row.SourceKey, csvFileBase, "/tmp/"+csvFileBase)
		if err != nil {
			printRow(row, err)
			return
		}
		err = shared.TransferFile(row.SourceUser, row.Source, ":22", row.SourceKey, exPath, "/tmp/"+exBaseName)
		if err != nil {
			printRow(row, err)
			return
		}
	}
	fmt.Println("running concurrent tests on " + row.Source)
	err = shared.RemoteExecSSH(row.SourceUser, row.Source, ":22", row.SourceKey, "chmod +x /tmp/"+exBaseName)
	if err != nil {
		printRow(row, err)
		return
	}
	if filter != nil && len(*filter) > 0 {
		err = shared.RemoteExecSSH(row.SourceUser, row.Source, ":22", row.SourceKey, "/tmp/"+exBaseName+" --file=/tmp/"+csvFileBase+" --filter='"+*filter+"'")
		if err != nil {
			printRow(row, err)
			return
		}
	} else {
		err = shared.RemoteExecSSH(row.SourceUser, row.Source, ":22", row.SourceKey, "/tmp/"+exBaseName+" --file=/tmp/"+csvFileBase)
		if err != nil {
			printRow(row, err)
			return
		}
	}
}
func writeStatus(lock *sync.RWMutex, index int, status string, statusMap map[int]string) {
	lock.Lock()
	defer lock.Unlock()
	statusMap[index] = status
}
func LocalExecCurrent(row *shared.Row, wg *sync.WaitGroup, index int, statusMap map[int]string, lock *sync.RWMutex) {
	defer wg.Done()
	timeout := time.Duration(row.TimeOut) * time.Millisecond
	d := net.Dialer{Timeout: timeout}
	protocol := strings.ToLower(row.Protocol)
	if protocol == "multicast" {
		protocol = "udp"
	}
	status := ""
	if !IsLocal(row.Source) {
		return
	}
	if strings.ToLower(row.Protocol) == "multicast" {
		err := shared.MulticastRead(row.TargetIP, row.TargetPort, timeout, false)
		if err != nil {
			status = "FAILED: " + err.Error()
		} else {
			status = "OK"
		}
	} else {
		conn, err := d.Dial(protocol, row.TargetIP+":"+row.TargetPort)
		if err != nil {
			status = "FAILED: " + err.Error()
		} else {
			conn.SetWriteDeadline(time.Now().Add(timeout))
			_, err = conn.Write([]byte("test\n"))
			if err != nil {
				status = "FAILED " + err.Error()
			} else {
				status = "WRITE OK "
				conn.SetReadDeadline(time.Now().Add(timeout))
				_, _, err = bufio.NewReader(conn).ReadLine()
				if err != nil {
					status = status + "/ READ FAILED " + err.Error()
				} else {
					status = "OK"
				}
			}
		}
		if conn != nil {
			_ = conn.Close()
		}
	}
	writeStatus(lock, index, status, statusMap)

}
func LocalExec(rows []shared.Row) {
	fmt.Printf("%-30s %-30s %-30s %-12s %-12s\n", "Description", "Source", "Target", "Protocol", "Status")
	fmt.Printf("--------------------------------------------------------------------------------------------------------------------------------\n")
	var wg sync.WaitGroup
	wg.Add(len(rows))
	statusMap := map[int]string{}
	lock := sync.RWMutex{}
	for i, _ := range rows {
		index := i
		currentRow := rows[index]
		go LocalExecCurrent(&currentRow, &wg, index, statusMap, &lock)
	}
	wg.Wait()
	for i, row := range rows {
		if status, ok := statusMap[i]; ok {
			fmt.Printf("%-30s %-30s %-30s %-12s %-12s\n", row.Description, hostName, row.TargetIP+":"+row.TargetPort, row.Protocol, status)
		}
	}
}
func main() {
	hostName, _ = os.Hostname()
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	filter := flag.String("filter", "", "filter rows by description")
	csvFile := flag.String("file", "", "csv file containing rules")
	remote := flag.Bool("remote", false, "run test on remote machines")
	overwrite := flag.Bool("overwrite", true, "overwrite rules file and itself on remote machines")
	flag.Parse()
	if len(*csvFile) == 0 {
		fmt.Println("Usage: testnetwork --file <csv file>")
		return
	}
	m, rows := ReadRows(*csvFile, filter)
	maxRoutines := GetMaxRoutines(m)
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
		var wg sync.WaitGroup
		wg.Add(maxRoutines)
		for _, k := range keys {
			row := m[k]
			if IsLocal(row.Source) {
				continue
			}
			go RemoteExec(*overwrite, &row, csvFileBase, ex, &wg, filter)
		}
		wg.Wait()
	} else {
		LocalExec(rows)
	}

}
