/*
DHCP Monitoring module

Tails syslog, watching for DHCP requests from MAC addresses known to be mobiles,
thus identifying if people are present on the network.
*/

package main

import (
	"database/sql"
	"github.com/ActiveState/tail"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strings"
	"time"
)

type Housemate struct {
	Id           int64
	Mac          string
	Name         string
	Last_seen    time.Time
	SeenDuration time.Duration
	IsHome       string
}

func (h *Housemate) isHome() bool {
	time_since_last_seen := time.Now().Sub(h.Last_seen)
	is_home := time_since_last_seen < (time.Minute * 10)
	return is_home
}

type DhcpStatus struct {
	db         *sql.DB
	housemates []*Housemate
	Last_ping  time.Time
}

func NewDhcpStatus(c *Config) *DhcpStatus {
	t := new(DhcpStatus)

	db, err := sql.Open("mysql", c.GetSqlURI())
	if err != nil {
		log.Println(err)
	}
	t.db = db

	return t
}

func (t *DhcpStatus) LastPersonActive() *Housemate {
	if len(t.housemates) == 0 {
		return nil
	}
	last_seen := t.housemates[0]
	for _, person := range t.housemates {
		if person.Last_seen.After(last_seen.Last_seen) {
			last_seen = person
		}
	}
	return last_seen
}

func (t *DhcpStatus) LoadMacs() error {
	t.housemates = make([]*Housemate, 0)

	rows, err := t.db.Query("SELECT id, mac, name from nest.people")
	if err != nil {
		log.Print(err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		h := new(Housemate)
		if err := rows.Scan(
			&h.Id,
			&h.Mac,
			&h.Name,
		); err != nil {
			continue
		}
		h.Last_seen = time.Now().Round(time.Second)
		t.housemates = append(t.housemates, h)
	}
	return nil
}

type SyslogLine struct {
	Timestamp time.Time
	Host      string
	Tag       string
	Message   string
}

func parseSyslogLine(line string) *SyslogLine {
	split_line := strings.Split(line, " ")
	l := new(SyslogLine)
	l.Timestamp = time.Now().Round(time.Second)
	l.Host = split_line[3]
	l.Tag = split_line[4]
	l.Message = strings.Join(split_line[4:], " ")
	return l

}

func (t *DhcpStatus) FollowLog() {
	for {
		// Open syslog for tailing
		tailer, err := tail.TailFile("/var/log/syslog", tail.Config{
			Follow: true,
			ReOpen: true,
			Poll:   true,
		})
		if err != nil {
			log.Println(err)
			return
		}

		// Iterate over tailed lines, check if they belong to dhcpd
		for line := range tailer.Lines {
			logline := parseSyslogLine(line.Text)
			// If the logline is DHCPD's, check it for known MACs
			if logline.Tag == "dhcpd:" {
				for _, housemate := range t.housemates {
					if strings.Contains(logline.Message, housemate.Mac) {
						housemate.Last_seen = logline.Timestamp
					}
				}
			}
		}
		log.Println("Tailer reloaded")
	}
}
