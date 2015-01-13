package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"time"
)

type Decider struct {
	db          *sql.DB
	dhcp_tailer *DhcpStatus
}

func NewDecider(c *Config, d *DhcpStatus) *Decider {
	t := new(Decider)

	db, err := sql.Open("mysql", c.GetSqlURI())
	if err != nil {
		log.Println(err)
	}
	t.db = db

	t.dhcp_tailer = d

	return t
}

func (d *Decider) getFloatSetting(name string) (float64, error) {
	row := d.db.QueryRow("SELECT value FROM nest.settings WHERE key = ?", name)
	var fv float64
	err := row.Scan(&fv)
	return fv, err
}

func (d *Decider) setFloatSetting(name string, value float64) error {
	_, err := d.db.Exec(
		"INSERT INTO nest.settings (key, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = ?",
		name, value, value,
	)
	return err
}

func (d *Decider) getBoolSetting(name string) (bool, error) {
	row := d.db.QueryRow("SELECT value FROM nest.settings WHERE key = ?", name)
	var bv bool
	err := row.Scan(&bv)
	return bv, err
}

func (d *Decider) setBoolSetting(name string, value bool) error {
	_, err := d.db.Exec(
		"INSERT INTO nest.settings (key, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = ?",
		name, value, value,
	)
	return err
}

func (d *Decider) getIdleTemp() float64 {
	// Grab the temperature to keep the house at when unoccupied
	temp, err := d.getFloatSetting("idle_temp")
	if err != nil {
		return 12.50
	}
	return temp
}

func (d *Decider) getActiveTemp() float64 {
	// Get the temperature to keep the house at when occupied
	temp, err := d.getFloatSetting("min_temp")
	if err != nil {
		return 15.50
	}
	return temp
}

func (d *Decider) getOverride() bool {
	// Return whether the furnace override is on
	override, err := d.getBoolSetting("override")
	if err != nil {
		return false
	}
	return override
}

func (d *Decider) anybodyHome() bool {
	last_seen := d.dhcp_tailer.LastPersonActive()
	if last_seen == nil {
		return false
	}
	time_since_last_seen := time.Now().Sub(last_seen.Last_seen)
	people_home := time_since_last_seen < (time.Minute * 10)
	return people_home
}

func (d *Decider) getLastFurnaceState() bool {
	// Return the state the furnace was in last time.
	// True = on, false = off
	state, err := d.getBoolSetting("furnace_on")
	if err != nil {
		return false
	}
	return state
}

func (d *Decider) getLastTemperature() float64 {
	row := d.db.QueryRow("SELECT temp  FROM `history` ORDER BY `timestamp` DESC LIMIT 1")
	var temp float64
	row.Scan(&temp)
	return temp
}

type HistData struct {
	Time     int64
	Temp     float64
	Pressure float64
}

func (d *Decider) getHistory() []*HistData {
	rows, err := d.db.Query(`
		SELECT timestamp, temp, pressure FROM nest.history WHERE
		timestamp > DATE_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 WEEK)
		AND id % 5 = 0
		ORDER BY timestamp ASC
	`)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	history := make([]*HistData, 0)
	for rows.Next() {
		h := new(HistData)
		var timestamp time.Time
		if err := rows.Scan(
			&timestamp,
			&h.Temp,
			&h.Pressure,
		); err != nil {
			continue
		}
		h.Time = timestamp.Unix()
		history = append(history, h)
	}
	return history
}

func (d *Decider) LogStats(current_temp, current_pressure float64, furnace_on bool) {
	_, err := d.db.Exec(`INSERT INTO  nest.history
		(id, timestamp, temp, pressure, heater, inhabited)
		VALUES
		(NULL, CURRENT_TIMESTAMP, ?, ?, ?, ?)`,
		current_temp, current_pressure, furnace_on, d.anybodyHome())
	if err != nil {
		log.Println(err)
	}

	err = d.setBoolSetting("furnace_on", furnace_on)
	if err != nil {
		log.Println(err)
	}
}

func (d *Decider) ShouldFurnace(current_temp float64) bool {
	// If the temp is lower than the idle temp, always turn up the heat
	if current_temp < d.getIdleTemp() {
		return true
	}

	// If people are home and the temp is below the active temp, turn on the heat
	if d.anybodyHome() && current_temp < d.getActiveTemp() {
		return true
	}

	// If the override is on, turn the furnace on no matter what
	if d.getOverride() {
		return true
	}

	// Sticky furnace on - don't toggle too frequently
	furnace_already_on := d.getLastFurnaceState()
	if furnace_already_on {
		if d.anybodyHome() {
			if current_temp < d.getActiveTemp()*1.05 {
				return true
			}
		} else {
			if current_temp < d.getIdleTemp()*1.05 {
				return true
			}
		}
	}

	return false
}
