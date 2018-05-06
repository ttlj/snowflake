package snowflake

import (
	"errors"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
)

// different functions to generate a machine ID

// EnvVarIPWorkerID returns lower 16 bits of an IP address
// passed over via environment variable MY_HOST_IP.
// Note: env var usage is meant for Kubernetes and its ilk
func EnvVarIPWorkerID() (uint16, error) {
	ipStr := os.Getenv("MY_HOST_IP")
	if len(ipStr) == 0 {
		return 0, errors.New("'MY_HOST_IP' environment variable not set")
	}
	ip := net.ParseIP(ipStr)
	if len(ip) < 4 {
		return 0, errors.New("invalid IP")
	}
	return uint16(ip[14])<<8 + uint16(ip[15]), nil
}

// K8sPodID returns lower 16 bits of an IP address
// passed over via environment variable MY_POD_NAME.
// Note: It requires StatefulSet pod names (my-pod-<number>)
func K8sPodID() (uint16, error) {
	pod := os.Getenv("MY_POD_NAME")
	if len(pod) == 0 {
		return 0, errors.New("'MY_POD_NAME' environment variable not set")
	}
	parts := strings.Split(pod, "-")
	i, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, errors.New("invalid Pod name")
	}
	if i > math.MaxUint16 {
		return 0, errors.New("Pod name contains too big number")
	}
	return uint16(i), nil
}

// Lower16BitPrivateIP returns lower 16 bits of a private IP
func Lower16BitPrivateIP() (uint16, error) {
	ip, err := privateIPv4()
	if err != nil {
		return 0, err
	}
	return uint16(ip[2])<<8 + uint16(ip[3]), nil
}

func privateIPv4() (net.IP, error) {
	as, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}

		ip := ipnet.IP.To4()
		if isPrivateIPv4(ip) {
			return ip, nil
		}
	}
	return nil, errors.New("no private ip address")
}

func isPrivateIPv4(ip net.IP) bool {
	return ip != nil &&
		(ip[0] == 10 || ip[0] == 172 && (ip[1] >= 16 && ip[1] < 32) || ip[0] == 192 && ip[1] == 168)
}
