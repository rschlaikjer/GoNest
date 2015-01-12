package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type WebServer struct {
	decider  *Decider
	servlets map[string]func(http.ResponseWriter, *http.Request)
}

func NewWebServer(d *Decider) *WebServer {
	t := new(WebServer)
	t.decider = d
	t.servlets = make(map[string]func(http.ResponseWriter, *http.Request))

	t.servlets["/nest.php"] = t.ControlPage
	return t
}

func (t *WebServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for path, servlet := range t.servlets {
		if strings.HasPrefix(r.RequestURI, path) {
			servlet(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func (t *WebServer) StatusPage(w http.ResponseWriter, r *http.Request) {

}

func (t *WebServer) ControlPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	current_temp_s := r.Form.Get("temp")
	current_temp, err := strconv.ParseFloat(current_temp_s, 64)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "burn-n")
		return
	}

	current_pressure_s := r.Form.Get("pressure")
	current_pressure, err := strconv.ParseFloat(current_pressure_s, 64)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "burn-n")
		return
	}

	furnace_on := t.decider.ShouldFurnace(current_temp)

	t.decider.LogStats(current_temp, current_pressure, furnace_on)

	if furnace_on {
		fmt.Fprintf(w, "burn-y")
	} else {
		fmt.Fprintf(w, "burn-n")
	}
}
