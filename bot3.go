package main

import (
	iniconf "code.google.com/p/goconf/conf"
	"encoding/json"
	"fmt"
	nsq "github.com/bitly/go-nsq"
	irc "github.com/fluffle/goirc/client"
	"github.com/gamelost/bot3server/server"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const BOT_CONFIG = "bot3.config"

// struct type for Bot3
type Bot3 struct {
	Config     *iniconf.ConfigFile
	Connection *irc.Conn
	// NSQ input/output to bot3Server
	BotServerOutputReader *nsq.Reader
	BotServerInputWriter  *nsq.Writer
	QuitChan              chan os.Signal
}

func (b *Bot3) init(config *iniconf.ConfigFile) error {

	b.Config = config

	// set up the config struct
	botNick, _ := b.Config.GetString("default", "nick")
	botPass, _ := b.Config.GetString("default", "pass")
	botServer, _ := b.Config.GetString("default", "ircserver")
	chanToJoin, _ := b.Config.GetString("default", "channel")
	bot3apiOutput, _ := b.Config.GetString("default", "bot3api-output")

	log.Printf("Bot nick will be: %s and will join %s\n", botNick, chanToJoin)
	cfg := irc.NewConfig(botNick)
	cfg.SSL = false
	cfg.Server = botServer
	cfg.NewNick = func(n string) string { return n + "^" }
	cfg.Pass = botPass
	c := irc.Client(cfg)

	// assign connection
	b.Connection = c

	// set up reader and message handler for botserver-output
	outputReader, err := nsq.NewReader(bot3apiOutput, "main")
	if err != nil {
		panic(err)
		b.QuitChan <- syscall.SIGINT
	}
	b.BotServerOutputReader = outputReader
	mh := &MessageHandler{Connection: c}
	b.BotServerOutputReader.AddHandler(mh)
	b.BotServerOutputReader.ConnectToLookupd("127.0.0.1:4161")

	// set up writer for botserver-input
	writer := nsq.NewWriter("127.0.0.1:4150")
	b.BotServerInputWriter = writer

	// Add handlers to do things here!
	// e.g. join a channel on connect.
	c.HandleFunc("connected",
		func(conn *irc.Conn, line *irc.Line) {
			log.Printf("Joining channel %s", chanToJoin)
			conn.Join(chanToJoin)
		})

	// And a signal on disconnect
	c.HandleFunc("disconnected",
		func(conn *irc.Conn, line *irc.Line) {
			log.Printf("Received quit command")
			b.QuitChan <- syscall.SIGINT
		})

	// hardcoded kill command just in case
	c.HandleFunc("PRIVMSG",
		func(conn *irc.Conn, line *irc.Line) {
			if strings.HasPrefix("!quit", line.Text()) {
				if line.Nick == "timzilla" {
					b.QuitChan <- syscall.SIGINT
				}
			}
		})

	// handle privmsgs
	c.HandleFunc("PRIVMSG",
		func(conn *irc.Conn, line *irc.Line) {

			botRequest := &server.BotRequest{RawLine: line}
			encodedRequest, _ := json.Marshal(botRequest)
			// write to nsq
			_, _, err := b.BotServerInputWriter.Publish("bot3api-input", encodedRequest)
			if err != nil {
				panic(err)
			}
		})

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

func main() {

	// the quit channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// read in necessary configuration
	config, err := iniconf.ReadConfigFile(BOT_CONFIG)
	if err != nil {
		log.Fatal("Unable to read configuration file. Exiting now.")
	}

	// set up Bot3 instnace
	bot3 := &Bot3{}
	bot3.QuitChan = sigChan
	bot3.init(config)
	bot3.connect()

	// receiving quit shuts down
	<-sigChan
}

type MessageHandler struct {
	Connection *irc.Conn
}

func (mh *MessageHandler) HandleMessage(message *nsq.Message) error {

	resp := &server.BotResponse{}
	json.Unmarshal(message.Body, resp)

	switch resp.ResponseType {
	case server.PRIVMSG:
		processPrivmsgResponse(mh.Connection, resp)
		break
	case server.ACTION:
		processActionResponse(mh.Connection, resp)
		break
	default:
		processPrivmsgResponse(mh.Connection, resp)
		break
	}

	return nil
}

func processPrivmsgResponse(conn *irc.Conn, botResponse *server.BotResponse) {

	for _, value := range botResponse.Response {
		conn.Privmsg(botResponse.Target, value)
	}
}

func processActionResponse(conn *irc.Conn, botResponse *server.BotResponse) {

	for _, value := range botResponse.Response {
		conn.Action(botResponse.Target, value)
	}
}
