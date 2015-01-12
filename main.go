package main

import (
	"log"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := LoadConfiguration("gonest.gcfg")

	dhcp_watcher := NewDhcpStatus(&config)
	dhcp_watcher.LoadMacs()
	go dhcp_watcher.FollowLog()
	for {
		time.Sleep(time.Second * 5)
	}
}
