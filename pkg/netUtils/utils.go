/**********************************************************/
/* Package netutils contain helper functions for network
/* connection
/**********************************************************/

package netutils

import (
	"net"
	"strings"
	"time"
)

var (
	tcpReadTimeoutMs = uint(0)
)

//Read function with timeout
func setReadTimeout(connRead net.Conn) error {
	if tcpReadTimeoutMs == 0 {
		return nil
	}

	tcpReadDeadline := time.Duration(tcpReadTimeoutMs) * time.Millisecond
	deadline := time.Now().Add(tcpReadDeadline)
	return connRead.SetReadDeadline(deadline)
}

//Return connection IP and port
func GetConnIp(c net.Conn) (string, string) {
	s := strings.Split(c.LocalAddr().String(), ":")
	ip := s[0]
	port := s[1]
	return ip, port
}
