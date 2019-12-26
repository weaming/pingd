// redis as storage backend
package redisHub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/weaming/pingd"
	ioRedis "github.com/weaming/pingd/io/redis"
)

type pingHTTP struct {
	startCh   chan<- pingd.HostStatus
	stopCh    chan<- pingd.HostStatus
	redisAddr string
	redisDB   int
	listKey   string
}

// ServeHTTP handles the incoming start/stop commands via HTTP
func (p pingHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Path[1:]
	if host == "" {
		w.WriteHeader(http.StatusBadRequest)
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
		_, hostname, _, err := ParseSchemeHostname(host)
		if err != nil {
			log.Println(err)
		}
		if strings.Contains(hostname, ":") {
			hostname = strings.Split(hostname, ":")[0]
		}

		err = checkDNS(hostname)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
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
