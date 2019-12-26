// redis as storage backend
package redisHub

import (
	"fmt"
	"log"
	"net/http"

	"github.com/weaming/pingd"
	"github.com/weaming/pingd/io/redis"
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
		fmt.Fprint(w, "missing host on request\n")
		return
	}

	connKV := redis.NewRedisConn(p.redisAddr, p.redisDB, "servehttp")
	switch r.Method {
	case "DELETE":
		fmt.Fprintf(w, "stop ping %s\n", host)
		redis.StopRedisHost(connKV, p.listKey, host, p.stopCh)
	default:
		fmt.Fprintf(w, "start ping %s\n", host)
		redis.StartRedisHost(connKV, p.listKey, host, p.startCh)
	}
}

// NewReceiverFunc returns the functions with sets up the system channels
// and starts the webserver, and listen on redis pubsub keys
func NewReceiverFunc(listen string, redisAddr string, redisDB int, startKey, stopKey, listKey string) pingd.Receiver {
	return func(startCh, stopCh chan<- pingd.HostStatus) {
		// redis receiver
		redisReceiver := redis.NewReceiverFunc(redisAddr, redisDB, startKey, stopKey, listKey)
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
