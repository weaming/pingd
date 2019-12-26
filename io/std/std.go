package std

import (
	"log"

	"github.com/weaming/pingd"
)

// NewLoaderFunc gets the sequence of hostnames/IPs and returns a Loader
// function that when called will insert them as Host structs into the
// start channel at boot time.
func NewLoaderFunc(hosts []string) pingd.Loader {
	return func(load chan<- pingd.HostStatus) {
		for _, host := range hosts {
			load <- pingd.HostStatus{Host: host, Down: false}
		}
	}
}

// NewNotifierFunc returns a function with just logs up and down events
func NewNotifierFunc() pingd.Notifier {
	return func(notifyCh <-chan pingd.HostStatus) {
		for {
			select {
			case h := <-notifyCh:
				switch h.Down {
				case true:
					log.Println("DOWN " + h.Host)
				case false:
					log.Println("UP " + h.Host)
				}
			}
		}
	}
}
