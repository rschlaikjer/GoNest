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
	Mac       string
	Name      string
	Last_seen time.Time
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

	rows, err := t.db.Query("SELECT mac, name from nest.people")
	if err != nil {
		log.Print(err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		h := new(Housemate)
		if err := rows.Scan(
			&h.Mac,
			&h.Name,
		); err != nil {
			continue
		}
		h.Last_seen = time.Now()
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
	l.Timestamp = time.Now()
	l.Host = split_line[3]
	l.Tag = split_line[4]
	l.Message = strings.Join(split_line[4:], " ")
	return l

}

func (t *DhcpStatus) FollowLog() {
	// Open syslog for tailing
	tailer, err := tail.TailFile("/var/log/syslog", tail.Config{
		Follow: true,
		ReOpen: true,
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
}
