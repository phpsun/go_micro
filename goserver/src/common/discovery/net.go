package discovery

import (
	"net"
	"strings"
)

func getListener() (listener net.Listener, host string, err error) {
	host = "0.0.0.0:0"
	listener, err = net.Listen("tcp", host)
	if err == nil {
		addr := listener.Addr().String()
		_, portString, _ := net.SplitHostPort(addr)
		host = strings.Replace(host, ":0", ":"+portString, 1)
	}
	return
}
