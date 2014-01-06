package main

import (
	"encoding/json"
	"github.com/bitly/go-nsq"
	"github.com/crosbymichael/ircstats"
	rethink "github.com/dancannon/gorethink"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type StoreHandler struct {
	session *rethink.Session
}

func cleanTableName(name string) string {
	name = strings.TrimLeft(name, "#")
	return strings.Replace(name, "-", "_", -1)
}

func newSession(addr, database string) (*rethink.Session, error) {
	return rethink.Connect(map[string]interface{}{
		"address":     addr,
		"database":    database,
		"maxIdle":     10,
		"idleTimeout": time.Second * 10,
	})
}

func (s *StoreHandler) HandleMessage(m *nsq.Message) error {
	var msg ircstats.Message
	if err := json.Unmarshal(m.Body, &msg); err != nil {
		return err
	}

	if _, err := rethink.Table(cleanTableName(msg.Channel)).Insert(msg).RunWrite(s.session); err != nil {
		return err
	}
	return nil
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	session, err := newSession("rethinkdb.docker:28015", "messages")
	if err != nil {
		log.Fatal(err)
	}

	reader, err := nsq.NewReader("messages", "store")
	if err != nil {
		log.Fatal(err)
	}
	reader.AddHandler(&StoreHandler{session})
	if err := reader.ConnectToLookupd("lookupd.docker:4161"); err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case <-reader.ExitChan:
			return
		case <-sigChan:
			reader.Stop()
		}
	}
}
