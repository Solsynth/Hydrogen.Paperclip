package gap

import "net"

func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return nil, err
	} else {
		defer conn.Close()
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}
