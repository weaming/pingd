// modified based on redis.go
package redisHub

import (
	"fmt"
	"log"

	"github.com/weaming/pingd"
	"github.com/weaming/pingd/io/redis"
)

const (
	upStatus   = "up"   // Status value for host up
	downStatus = "down" // Status value for host down

	// when receiving host on the start channel
	// they can be requested to start as "down"
	// adding it at the end eg "example.com down"
	downSuffix = " down"
)

// NewNotifierFunc returns the function that
// publishes on redis the up/down events
func NewNotifierFunc(redisAddr string, redisDB int, upKey, downKey, topicPrefix string) pingd.Notifier {
	return func(notifyCh <-chan pingd.HostStatus) {
		conn := redis.NewRedisConn(redisAddr, redisDB, "notify")

		var h pingd.HostStatus
		for {
			select {
			case h = <-notifyCh:
				topics := []string{"global", topicPrefix, topicPrefix + "/" + h.Host}

				switch h.Down {
				// DOWN
				case true:
					log.Println("DOWN " + h.Host)
					conn.Send("PUBLISH", downKey, fmt.Sprintf("%s %s", h.Host, h.Reason))
					conn.Send("SET", "status-"+h.Host, downStatus)
					conn.Flush()

					// hub
					PostToHub(NewPubMessage(TYPE_PLAIN, fmt.Sprintf("DOWN %s: %s", h.Host, h.Reason), topics))
				// UP
				case false:
					log.Println("UP " + h.Host)
					conn.Send("PUBLISH", upKey, h.Host)
					conn.Send("SET", "status-"+h.Host, upStatus)
					conn.Flush()

					// hub
					PostToHub(NewPubMessage(TYPE_PLAIN, "UP "+h.Host, topics))
				}
			}
		}
	}
}
