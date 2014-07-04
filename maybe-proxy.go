package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
)

const (
	usage = `usage: maybe-proxy <host> <port>`
)

var (
	proxy   string
	noProxy []*net.IPNet
)

func init() {
	log.SetFlags(0)
	if str := os.Getenv("MAYBE_PROXY"); str != "" {
		proxy = str
	}
	dontProxy("0.0.0.0/8")
	dontProxy("10.0.0.0/8")
	dontProxy("127.0.0.0/8")
	dontProxy("169.254.0.0/16")
	dontProxy("172.16.0.0/12")
	dontProxy("192.168.0.0/16")
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf(usage)
	}
	host := os.Args[1]
	port := os.Args[2]

	nc, err := exec.LookPath("nc")
	if err != nil {
		log.Fatal(err)
	}

	if shouldProxy(host, port) {
		log.Printf("Proxying connection to %s:%s", host, port)
		err = syscall.Exec(nc, []string{"nc", "-X", "connect", "-x", proxy, host, port}, os.Environ())
	} else {
		err = syscall.Exec(nc, []string{"nc", host, port}, os.Environ())
	}
	if err != nil {
		log.Fatal(err)
	}
}

func dontProxy(cidr string) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Fatal(err)
	}
	noProxy = append(noProxy, network)
}

func onVPN() bool {
	ifs, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range ifs {
		// TODO: There's got to be a more "correct" way to do this.
		mask := net.FlagUp | net.FlagPointToPoint
		if i.Flags&mask == mask && i.Name == "utun0" {
			return true
		}
	}
	return false
}

func shouldProxy(host, port string) bool {
	if !onVPN() {
		return false
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		log.Fatal(err)
	}
	for _, ip := range ips {
		if len(ip) != 4 {
			// IPv6
			continue
		}
		for _, network := range noProxy {
			if network.Contains(ip) {
				return false
			}
		}
	}
	return true
}
