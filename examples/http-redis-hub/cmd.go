package main

import (
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/weaming/pingd"
	"github.com/weaming/pingd/io/redis"
	"github.com/weaming/pingd/io/redisHub"
	"github.com/weaming/pingd/ping"
)

// See flags
var (
	redisAddr      string
	redisDB        int
	failLimit      int
	interval       time.Duration
	listenAddr     string
	hubTopicPrefix string
)

func main() {
	flag.StringVar(&redisAddr, "redis", ":6379", "Redis IP:port")
	flag.IntVar(&redisDB, "redisDB", 0, "Redis DB [0..15]")
	flag.IntVar(&failLimit, "failLimit", 6, "number failed ping attempts in a row to consider host down")
	flag.DurationVar(&interval, "interval", 10*time.Second, "seconds between each ping")
	flag.DurationVar(&ping.TimeOut, "timeOut", 5*time.Second, "seconds for single ping timeout")
	flag.StringVar(&listenAddr, "listen", ":8080", "webserver listen address")
	flag.StringVar(&hubTopicPrefix, "hubTopic", "admin/ping", "Topic for https://hub.drink.cafe")
	flag.Parse()

	var pool = &pingd.Pool{
		Interval:  interval,
		FailLimit: failLimit,
		Ping:      redisHub.Ping,
		Receive:   redisHub.NewReceiverFunc(listenAddr, redisAddr, redisDB, "pingStart", "pingStop", "pingHostList"),
		Notify:    redisHub.NewNotifierFunc(redisAddr, redisDB, "up", "down", hubTopicPrefix),
		Load:      redis.NewLoaderFunc(redisAddr, redisDB, "pingHostList"),
	}

	pool.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c // Exit on interrupt
}
