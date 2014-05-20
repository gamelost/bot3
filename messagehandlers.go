package main

import (
	"encoding/json"
	nsq "github.com/bitly/go-nsq"
	"github.com/gamelost/bot3server/server"
	irc "github.com/gamelost/goirc/client"
)

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

type HeartbeatMessageHandler struct {
	Bot3ServerHeartbeatChan chan *server.Bot3ServerHeartbeat
}

func (mh *HeartbeatMessageHandler) HandleMessage(message *nsq.Message) error {

	hb := &server.Bot3ServerHeartbeat{}
	json.Unmarshal(message.Body, hb)
	mh.Bot3ServerHeartbeatChan <- hb
	return nil
}
