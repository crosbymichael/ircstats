package ircstats

import "time"

type Message struct {
	Nick      string    `json:"nick"`
	Message   string    `json:"message"`
	Channel   string    `json:"channel"`
	Timestamp time.Time `json:"timestamp"`
}
