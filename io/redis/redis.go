package redis

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/weaming/pingd"
)

const (
	upStatus   = "up"   // Status value for host up
	downStatus = "down" // Status value for host down

	// when receiving host on the start channel
	// they can be requested to start as "down"
	// adding it at the end eg "example.com down"
	downSuffix = " down"
)

func StartRedisHost(connKV redis.Conn, listKey, host string, startHostCh chan<- pingd.HostStatus) {
	down := false
	if strings.HasSuffix(host, downSuffix) {
		down = true
		host = strings.Replace(host, downSuffix, "", 1)
	}

	// Add to the list of pinged hosts
	_, err := connKV.Do("SADD", listKey, host)
	if err != nil {
		log.Panicln(err)
	}
	connKV.Flush()
	startHostCh <- pingd.HostStatus{Host: host, Down: down}
}

func StopRedisHost(connKV redis.Conn, listKey, host string, stopHostCh chan<- pingd.HostStatus) {
	// Remove from the list of pinged hosts
	_, err := connKV.Do("SREM", listKey, host)
	if err != nil {
		log.Panicln(err)
	}
	connKV.Flush()
	stopHostCh <- pingd.HostStatus{Host: host}
}

func NewRedisConn(redisAddr string, redisDB int, purpose string) redis.Conn {
	conn, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		log.Panicln(err)
	}
	_, err = conn.Do("SELECT", redisDB)
	if err != nil {
		log.Panicln(err)
	}

	servername, _ := os.Hostname()
	_, err = conn.Do("CLIENT", "SETNAME", purpose+"-"+servername)
	if err != nil {
		log.Panicln(err)
	}
	return conn
}

// NewReceiverFunc returns the function that
// listens of redis for start/stop commands
func NewReceiverFunc(redisAddr string, redisDB int, startKey, stopKey, listKey string) pingd.Receiver {
	return func(startHostCh, stopHostCh chan<- pingd.HostStatus) {
		connKV := NewRedisConn(redisAddr, redisDB, "receive-kv")
		conPubSub := NewRedisConn(redisAddr, redisDB, "receive-pubsub")

		psc := redis.PubSubConn{conPubSub}
		psc.Subscribe(startKey, stopKey)

		for {
			switch n := psc.Receive().(type) {
			case redis.Message:
				if n.Channel == startKey {
					host := string(n.Data)
					StartRedisHost(connKV, listKey, host, startHostCh)

				} else if n.Channel == stopKey {
					host := string(n.Data)
					StopRedisHost(connKV, listKey, host, startHostCh)
				}

			case redis.PMessage:
			case redis.Subscription:
				log.Println("BOOT Listening to " + n.Channel)
			case error:
				log.Printf("error: %v\n", n)
				return
			}
		}
	}
}

// NewNotifierFunc returns the function that
// publishes on redis the up/down events
func NewNotifierFunc(redisAddr string, redisDB int, upKey, downKey string) pingd.Notifier {
	return func(notifyCh <-chan pingd.HostStatus) {
		conn := NewRedisConn(redisAddr, redisDB, "notify")

		var h pingd.HostStatus
		for {
			select {
			case h = <-notifyCh:
				switch h.Down {
				// DOWN
				case true:
					log.Println("DOWN " + h.Host)
					conn.Send("PUBLISH", downKey, fmt.Sprintf("%s %s", h.Host, h.Reason))
					conn.Send("SET", "status-"+h.Host, downStatus)
					conn.Flush()
				// UP
				case false:
					log.Println("UP " + h.Host)
					conn.Send("PUBLISH", upKey, h.Host)
					conn.Send("SET", "status-"+h.Host, upStatus)
					conn.Flush()
				}
			}
		}
	}
}

func LoadStatus(conn redis.Conn, redisKey string) []pingd.HostStatus {
	hosts, err := redis.Strings(conn.Do("SMEMBERS", redisKey))
	if err != nil {
		log.Panicln(err)
	}

	statuses := []pingd.HostStatus{}
	for _, host := range hosts {
		var down bool
		status, err := redis.String(conn.Do("GET", "status-"+host))
		if err != nil {
			log.Println("ERROR loading status of " + host + ". Assuming UP")
		}
		if status == downStatus {
			down = true
		}
		statuses = append(statuses, pingd.HostStatus{Host: host, Down: down})
	}
	return statuses
}

// NewLoaderFunc returns the function that loads back
// hosts and last statuses from REDIS in case of reboot
// send them to the startHostCh channel
func NewLoaderFunc(redisAddr string, redisDB int, listKey string) pingd.Loader {
	return func(startHostCh chan<- pingd.HostStatus) {
		log.Println("BOOT Loading hosts")
		conn := NewRedisConn(redisAddr, redisDB, "load")
		statuses := LoadStatus(conn, listKey)

		for _, status := range statuses {
			// load into process
			startHostCh <- status
			// slow a bit loading process
			time.Sleep(time.Millisecond * 10)
		}

		log.Println("BOOT " + strconv.Itoa(len(statuses)) + " hosts loaded")
		log.Println("BOOT Ready")
	}

}
