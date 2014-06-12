package main

import (
	"code.google.com/p/go-uuid/uuid"
	iniconf "code.google.com/p/goconf/conf"
	"encoding/json"
	"fmt"
	"github.com/gamelost/bot3server/server"
	nsq "github.com/gamelost/go-nsq"
	irc "github.com/gamelost/goirc/client"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	BOT_CONFIG = "bot3.config"
	PRIVMSG    = "PRIVMSG"
)

type Request struct {
	RawLine   *irc.Line
	Timestamp time.Time
}

func main() {

	// the quit channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// read in necessary configuration
	configFile, err := iniconf.ReadConfigFile(BOT_CONFIG)
	if err != nil {
		log.Fatal("Unable to read configuration file. Exiting now.")
	}

	// convert to bot3config
	bot3config, err := Bot3ConfigFromConfigFile(configFile)
	if err != nil {
		log.Fatal("Incomplete or invalid config file for bot3config. Exiting now.")
	}

	// set up Bot3 instnace
	bot3 := &Bot3{}
	bot3.QuitChan = sigChan
	bot3.init(bot3config)
	bot3.connect()

	// receiving quit shuts down
	<-sigChan
}

// struct type for Bot3
type Bot3 struct {
	Config     *Bot3Config
	Connection *irc.Conn
	// NSQ input/output to bot3Server
	BotServerOutputReader    *nsq.Reader
	BotServerInputWriter     *nsq.Writer
	BotServerHeartbeatReader *nsq.Reader
	// program quit chan
	QuitChan chan os.Signal
	// bot state
	Silenced                bool
	BotServerOnline         bool
	LastBotServerHeartbeat  *server.Bot3ServerHeartbeat
	Bot3ServerHeartbeatChan chan *server.Bot3ServerHeartbeat
	MagicIdentifier         string
	// messagehandlers
	IRCMessageHandler *MessageHandler
}

func (b *Bot3) init(config *Bot3Config) error {

	b.Config = config
	b.Silenced = false

	// set up magicid
	b.MagicIdentifier = config.MagicIdentifier

	log.Printf("Bot nick will be: %s and will join %s\n", b.Config.BotNick, b.Config.BotChannelToJoin)
	cfg := irc.NewConfig(b.Config.BotNick)
	cfg.SSL = false
	cfg.Server = b.Config.BotIRCServer
	cfg.NewNick = func(n string) string { return n + "^" }
	cfg.Pass = b.Config.BotPass
	c := irc.Client(cfg)

	// assign connection
	b.Connection = c
	b.Bot3ServerHeartbeatChan = make(chan *server.Bot3ServerHeartbeat)
	b.BotServerOnline = true

	// set up listener for heartbeat from bot3server
	heartbeatReader, err := nsq.NewReader("bot3server-heartbeat", "main#ephemeral")
	if err != nil {
		panic(err)
		b.QuitChan <- syscall.SIGINT
	}
	b.BotServerHeartbeatReader = heartbeatReader
	hbmh := &HeartbeatMessageHandler{Bot3ServerHeartbeatChan: b.Bot3ServerHeartbeatChan}
	b.BotServerHeartbeatReader.AddHandler(hbmh)
	b.BotServerHeartbeatReader.ConnectToLookupd("127.0.0.1:4161")

	// set up goroutine to listen for heartbeat
	go func() {
		for {
			select {
			case <-b.Bot3ServerHeartbeatChan:
				// if we're coming back online, broadcast message
				if b.BotServerOnline == false {
					b.BotServerOnline = true
					c.Nick(b.Config.BotNick)
					c.Privmsg(b.Config.BotChannelToJoin, "Wheee.  Restored connection to bot3server!")
				}
				break
			case <-time.After(time.Second * 5):
				// if initially going offline, broadcast message
				if b.BotServerOnline == true {
					c.Privmsg(b.Config.BotChannelToJoin, "Welp! I seem to have lost connection to the bot3server.  Will continue to look for it.")
					c.Nick(b.Config.BotOfflinePrefix + "_SERVEROFFLINE")
					b.BotServerOnline = false
				}
				break
			}
		}
	}()

	// set up reader and message handler for botserver-output
	outputReader, err := nsq.NewReader(b.Config.Bot3ServerOutputTopic, "main")
	if err != nil {
		panic(err)
		b.QuitChan <- syscall.SIGINT
	}
	b.BotServerOutputReader = outputReader
	b.IRCMessageHandler = &MessageHandler{Connection: c, MagicIdentifier: b.Config.MagicIdentifier, Requests: map[string]*Request{}}
	b.BotServerOutputReader.AddHandler(b.IRCMessageHandler)
	b.BotServerOutputReader.ConnectToLookupd("127.0.0.1:4161")

	// set up writer for botserver-input
	writer := nsq.NewWriter("127.0.0.1:4150")
	b.BotServerInputWriter = writer

	// Add handlers to do things here!
	// e.g. join a channel on connect.
	c.HandleFunc("connected",
		func(conn *irc.Conn, line *irc.Line) {
			log.Printf("Joining channel %s", b.Config.BotChannelToJoin)
			conn.Join(b.Config.BotChannelToJoin)
		})

	// And a signal on disconnect
	c.HandleFunc("disconnected",
		func(conn *irc.Conn, line *irc.Line) {
			log.Printf("Received quit command")
			b.QuitChan <- syscall.SIGINT
		})

	// hardcoded kill command just in case
	c.HandleFunc(PRIVMSG,
		func(conn *irc.Conn, line *irc.Line) {
			if strings.HasPrefix("!quit "+b.Config.BotNick, line.Text()) {
				if b.Config.IsAdminNick(line.Nick) {
					b.QuitChan <- syscall.SIGINT
				}
			}
		})

	// handle !silent
	c.HandleFunc(PRIVMSG,
		func(conn *irc.Conn, line *irc.Line) {
			if strings.HasPrefix("!silent", line.Text()) {
				if b.Config.IsAdminNick(line.Nick) {
					if !b.Silenced {
						c.Nick(b.Config.BotOfflinePrefix + "_SILENCED")
						b.Silenced = true
					} else {
						c.Nick(b.Config.BotNick)
						b.Silenced = false
					}
				}
			}
		})

	// handle privmsgs
	c.HandleFunc(PRIVMSG,
		func(conn *irc.Conn, line *irc.Line) {
			reqid := uuid.NewUUID().String()
			b.IRCMessageHandler.Requests[reqid] = &Request{RawLine: line, Timestamp: time.Now()}
			botRequest := &server.BotRequest{Identifier: reqid, Nick: line.Nick, Channel: line.Target(), ChatText: line.Text()}
			encodedRequest, err := json.Marshal(botRequest)

			if err != nil {
				log.Printf("Error encoding request: %s\n", err.Error())
			}

			// write to nsq only if not silenced, otherwise drop message
			if !b.Silenced {
				_, _, err := b.BotServerInputWriter.Publish(b.Config.Bot3ServerInputTopic, encodedRequest)
				if err != nil {
					panic(err)
				}
			} else {
				log.Printf("Silenced - will not output message.")
			}
		})

	// disabling this for now since it breaks remindme
	// go func() {
	// 	for {
	// 		exp := time.Now()
	// 		<-time.After(time.Second * 5)
	// 		for k, v := range b.IRCMessageHandler.Requests {
	// 			if v.Timestamp.Unix() < exp.Unix() {
	// 				log.Printf("Expiring %+v\n", v)
	// 				delete(b.IRCMessageHandler.Requests, k)
	// 			}
	// 		}
	// 	}
	// }()

	return nil
}

func (b *Bot3) connect() {

	// Tell client to connect.
	if err := b.Connection.Connect(); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
		b.QuitChan <- syscall.SIGINT
	} else {
		log.Printf("Successfully connected.")
	}

}
