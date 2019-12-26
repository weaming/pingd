// redis as storage backend
package redisHub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"time"

	"github.com/weaming/pingd"
	ioRedis "github.com/weaming/pingd/io/redis"
	"github.com/weaming/pingd/ping"
)

type pingHTTP struct {
	startCh   chan<- pingd.HostStatus
	stopCh    chan<- pingd.HostStatus
	redisAddr string
	redisDB   int
	listKey   string
}

func (p pingHTTP) HandlerProtocol(protocol string, fn func(chan<- pingd.HostStatus) error) {
}

// ServeHTTP handles the incoming start/stop commands via HTTP
func (p pingHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Path[1:]
	if host == "" {
		fmt.Fprint(w, "missing host on request\n")
		return
	}

	if host == "status" {
		p.serveStatus(w, r)
		return
	}

	connKV := ioRedis.NewRedisConn(p.redisAddr, p.redisDB, "servehttp")
	switch r.Method {
	case "DELETE":
		fmt.Fprintf(w, "stop ping %s\n", host)
		ioRedis.StopRedisHost(connKV, p.listKey, host, p.stopCh)
	default:
		err := checkDNS(host)
		if err != nil {
			fmt.Fprint(w, err.Error())
			return
		}
		fmt.Fprintf(w, "start ping %s\n", host)
		ioRedis.StartRedisHost(connKV, p.listKey, host, p.startCh)
	}
}

func (p pingHTTP) serveStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	conn := ioRedis.NewRedisConn(p.redisAddr, p.redisDB, "status")
	statuses := ioRedis.LoadStatus(conn, p.listKey)
	json.NewEncoder(w).Encode(map[string]interface{}{"data": statuses})
}

// NewReceiverFunc returns the functions with sets up the system channels
// and starts the webserver, and listen on redis pubsub keys
func NewReceiverFunc(listen string, redisAddr string, redisDB int, startKey, stopKey, listKey string) pingd.Receiver {
	return func(startCh, stopCh chan<- pingd.HostStatus) {
		// redis receiver
		redisReceiver := ioRedis.NewReceiverFunc(redisAddr, redisDB, startKey, stopKey, listKey)
		go redisReceiver(startCh, stopCh)

		// http receiver
		var p = &pingHTTP{startCh, stopCh, redisAddr, redisDB, listKey}
		log.Printf("Web server starting on %s", listen)
		err := http.ListenAndServe(listen, p)
		if err != nil {
			log.Fatal(err)
		}

	}
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	// https://golang.org/src/net/http/transport.go

	tr := &http.Transport{
		// MaxIdleConnsPerHost, if non-zero, controls the maximum idle (keep-alive) connections to keep per-host.
		// If zero, DefaultMaxIdleConnsPerHost is used, whose value is 2.
		MaxIdleConnsPerHost: 1024,
		// MaxIdleConns controls the maximum number of idle (keep-alive) connections across all hosts.
		// Zero means no limit.
		MaxIdleConns:          1000,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   timeout * time.Second,
	}
}

var httpGetClient = NewHTTPClient(10)

func Ping(host string) (up bool, err error) {
	if strings.HasPrefix(strings.ToLower(host), "http") {
		// log.Printf("GET %s", host)
		resp, err := httpGetClient.Get(host)
		if resp.StatusCode < 500 {
			return true, err
		} else {
			return false, fmt.Errorf("status code is %d", resp.StatusCode)
		}
	} else {
		return ping.Ping(host)
	}
}
