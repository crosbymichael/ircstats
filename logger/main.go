package main

import (
	"encoding/json"
	"flag"
	"github.com/bitly/go-nsq"
	"github.com/crosbymichael/ircstats"
	irc "github.com/thoj/go-ircevent"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	server   string
	channels []string
	nick     string
	verbose  bool
)

func init() {
	flag.StringVar(&server, "s", "", "IRC server url")
	flag.StringVar(&nick, "nick", "", "IRC nick")
	flag.BoolVar(&verbose, "v", false, "Verbose ouput of requests")

	flag.Parse()

	if nick == "" {
		log.Fatal("Nick must be specified")
	}
	if server == "" {
		log.Fatal("Server must be specified")
	}

	for _, arg := range flag.Args() {
		if arg[0] != '#' {
			log.Fatalf("Channel %s should start with #", arg)
		}
		channels = append(channels, arg)
	}
	if len(channels) == 0 {
		log.Fatal("No channels specified")
	}
}

func producer(c chan *ircstats.Message, group *sync.WaitGroup) {
	group.Add(1)
	defer group.Done()

	writer := nsq.NewWriter("nsqd.docker:4150")
	defer writer.Stop()

	for msg := range c {
		data, err := json.Marshal(msg)
		if err != nil {
			log.Fatal(err)
		}

		// Passing nil to the doneChan because we do not care about the response right now
		if err := writer.PublishAsync("messages", data, nil); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var (
		c       = make(chan *ircstats.Message)
		conn    = irc.IRC(nick, nick)
		group   = &sync.WaitGroup{}
		sigChan = make(chan os.Signal, 1)
		done    = make(chan int)
	)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	conn.VerboseCallbackHandler = verbose

	go producer(c, group)

	conn.AddCallback("001", func(e *irc.Event) {
		for _, channel := range channels {
			conn.Join(channel)
			time.Sleep(1 * time.Second)
		}
	})

	conn.AddCallback("PRIVMSG", func(e *irc.Event) {
		c <- &ircstats.Message{
			Nick:      e.Nick,
			Message:   e.Message,
			Channel:   e.Arguments[0],
			Timestamp: e.Timestamp,
		}
	})

	if err := conn.Connect(server); err != nil {
		log.Fatal(err)
	}

	go func() {
		conn.Loop()
		done <- 0
	}()

	for {
		select {
		case <-done:
			close(c)
			group.Wait()
			return
		case <-sigChan:
			conn.Quit()
		}
	}
}
