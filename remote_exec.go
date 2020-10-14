package remotejob

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const hostsFileName = "hosts.conf"
const defaultUser = "root"
const defaultPass = "root"
const defaultCmd = "shutdown -h now"

var timeout, _ = time.ParseDuration("5s")

// Host information wrapper
type Host struct {
	Name     string
	IP       string
	User     string
	Password string
	CMD      string
}

func (h *Host) String() string {
	return fmt.Sprintf("Name: %s, IP: %s, User: %s, Password: %s, CMD: %s", h.Name, h.IP, h.User, h.Password, h.CMD)
}

// NewHost create a new host with default user, pass and command
func NewHost(name, ip string) *Host {
	return &Host{name, ip, defaultUser, defaultPass, defaultCmd}
}

// NewHostFromRecord create a host from a comma string array
func NewHostFromRecord(record []string) (*Host, error) {
	items := len(record)
	if record == nil || items < 2 {
		return nil, fmt.Errorf("Empty or not valid record: %s", record)
	}
	host := NewHost(strings.TrimSpace(record[0]), strings.TrimSpace(record[1]))
	if items >= 3 {
		host.User = strings.TrimSpace(record[2])
	}
	if items >= 4 {
		host.Password = strings.TrimSpace(record[3])
	}
	if items >= 5 {
		host.CMD = strings.TrimSpace(strings.Join(record[4:], ","))
	}
	return host, nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// ReadFromFile read conf files and create hosts
func ReadFromFile(filePath *string) []*Host {
	file, err := os.Open(*filePath)
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	hosts := make([]*Host, 0)
	for scanner.Scan() {
		record := strings.TrimSpace(scanner.Text())
		// skip comments
		if strings.HasPrefix(record, "#") {
			continue
		}
		records := strings.Split(record, ",")
		host, err := NewHostFromRecord(records)
		if err == nil {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

// ExecuteCmd connect to remote host and execute the command
func ExecuteCmd(host *Host, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	sshConfig := &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(host.Password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = host.Password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.IP, 22), sshConfig)
	if err != nil {
		fmt.Printf("%s> %s\n", host.IP, err)
		return
	}
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("%s> %s\n", host.IP, err)
		client.Close()
		return
	}
	session.Run(host.CMD)
	fmt.Printf("Executed: %s (%s) [%s]\n", host.Name, host.IP, host.CMD)
	session.Close()
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func main() {
	hostFile := flag.String("f", hostsFileName, "The conf file")
	flag.Parse()
	hosts := ReadFromFile(hostFile)
	var wg sync.WaitGroup
	var localhost *Host = nil
	localip, err := externalIP()
	for _, host := range hosts {
		if err == nil && (host.IP == localip || host.IP == "localhost" || host.IP == "0.0.0.0" || host.IP == "127.0.0.1") {
			localhost = host
			continue
		}
		wg.Add(1)
		go ExecuteCmd(host, &wg)
	}
	wg.Wait()
	if localhost != nil {
		fmt.Println("Executing on localhost...")
		ExecuteCmd(localhost, nil)
	}
	fmt.Println("All task executed.")
}
