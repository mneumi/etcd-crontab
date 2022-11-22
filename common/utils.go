package common

import (
	"errors"
	"net"
	"strings"
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

func DeleteElement(s []string, elem string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == elem {
			s = append(s[:i], s[i+1:]...)
			i--
		}
	}
	return s
}

func ExtraceJobNameByKey(key string) string {
	return strings.TrimPrefix(key, JOB_SAVE_DIR)
}

func ExtraceWorkerIDByKey(key string) string {
	return strings.TrimPrefix(key, WORKER_DIR)
}

func ExtractInterceptNameByKey(key string) string {
	return strings.TrimPrefix(key, INTERCEPT_DIR)
}
