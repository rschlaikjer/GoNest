package main

import (
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := LoadConfiguration("gonest.gcfg")

	dhcp_watcher := NewDhcpStatus(config)
	dhcp_watcher.LoadMacs()
	go dhcp_watcher.FollowLog()

	decider := NewDecider(config, dhcp_watcher)

	webserver := NewWebServer(decider)

	bind_address := config.Network.BindAddress + ":" + config.Network.BindPort
	http_server := http.Server{
		Addr:           bind_address,
		Handler:        webserver,
		MaxHeaderBytes: 1 << 20,
	}

	if err := http_server.ListenAndServe(); err != nil {
		log.Fatalln("Fatal Error: ListenAndServe: ", err.Error())
	}

}
