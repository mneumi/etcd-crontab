package common

import (
	"errors"
	"math/rand"
	"net"
	"time"
)

func GetIPv4() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", errors.New("没有找到网卡的IPv4地址")
}

func RandString(len int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}

func DeleteSliceByElement(s []string, elem string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == elem {
			s = append(s[:i], s[i+1:]...)
			i--
		}
	}
	return s
}
