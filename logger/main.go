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
)

var (
	server  string
	channel string
	nick    string
	verbose bool
	nsqd    string
)

func init() {
	flag.StringVar(&server, "s", "", "IRC server url")
	flag.StringVar(&nick, "nick", "", "IRC nick")
	flag.BoolVar(&verbose, "v", false, "Verbose ouput of requests")

	flag.Parse()

	if channel = flag.Arg(0); channel == "" {
		log.Fatalln("Channel must be specified")
	}
	if nick == "" {
		log.Fatalln("Nick must be specified")
	}
	if server == "" {
		log.Fatalln("Server must be specified")
	}
	nsqd = "nsqd.docker:4150"
	/*
		if nsqd = ircstats.GetLink("NSQD", "4150"); nsqd == "" {
			log.Fatalln("Nsqd must be linked")
		}
	*/
}

func producer(c chan *ircstats.Message, group *sync.WaitGroup) {
	group.Add(1)
	defer group.Done()

	writer := nsq.NewWriter(nsqd)
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
		conn.Join(channel)
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
