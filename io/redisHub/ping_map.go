package redisHub

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/weaming/pingd/ping"
)

var httpGetClient = NewHTTPClient(10)

type PingMap struct {
	timeout   time.Duration
	schemeMap map[string]func(string, time.Duration) (bool, error)
}

func NewPingMap(timeout time.Duration) PingMap {
	return PingMap{
		timeout,
		map[string]func(string, time.Duration) (bool, error){
			"http":  PingHTTP,
			"https": PingHTTP,
			"telnet": func(host string, timeout time.Duration) (bool, error) {
				_, hostname, port, _ := ParseSchemeHostname(host)
				err := rawConnect(hostname, port, timeout)
				return err != nil, err
			},
		},
	}
}

func (p PingMap) Ping(host string) (up bool, err error) {
	scheme, _, _, err := ParseSchemeHostname(host)
	// log.Printf("ping %s with scheme %s\n", host, scheme)
	if fn, ok := p.schemeMap[scheme]; ok {
		return fn(host, p.timeout)
	}
	return ping.Ping(host)
}

func PingHTTP(host string, timeout time.Duration) (up bool, err error) {
	resp, err := NewHTTPClient(timeout).Get(host)
	if err != nil {
		return false, err
	}
	if resp.StatusCode < 500 {
		return true, err
	} else {
		return false, fmt.Errorf("status code is %d", resp.StatusCode)
	}
}

func rawConnect(host string, port string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		// fmt.Println("Connecting error:", err)
		return err
	}
	if conn != nil {
		defer conn.Close()
		// fmt.Println("Opened", net.JoinHostPort(host, port))
	}
	return nil
}

func ParseSchemeHostname(host string) (string, string, string, error) {
	var u *url.URL
	var err error
	// treat google.com:443 style as telnet
	if !strings.Contains(host, "://") {
		if strings.Contains(host, ":") {
			fakeHost := "telnet://" + host
			u, err = url.Parse(fakeHost)
		} else {
			fakeHost := "icmp://" + host
			u, err = url.Parse(fakeHost)
		}
	} else {
		u, err = url.Parse(host)
	}
	if err != nil {
		log.Printf("parse url failed: %s\n", err)
		return "", host, "", err
	}
	return u.Scheme, u.Hostname(), u.Port(), nil
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	// https://golang.org/src/net/http/transport.go

	tr := &http.Transport{
		MaxIdleConnsPerHost: 1000,
		MaxIdleConns:        1000,
		IdleConnTimeout:     60 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   timeout * time.Second,
	}
}
