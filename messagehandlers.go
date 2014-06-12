package main

import (
	"encoding/json"
	"github.com/gamelost/bot3server/server"
	nsq "github.com/gamelost/go-nsq"
	irc "github.com/gamelost/goirc/client"
	"log"
)

type MessageHandler struct {
	Connection      *irc.Conn
	Requests        map[string]*Request
	MagicIdentifier string
}

func (mh *MessageHandler) HandleMessage(message *nsq.Message) error {

	resp := &server.BotResponse{}
	json.Unmarshal(message.Body, resp)

	switch resp.ResponseType {
	case server.PRIVMSG:
		mh.processPrivmsgResponse(resp)
		break
	case server.ACTION:
		mh.processActionResponse(resp)
		break
	default:
		mh.processPrivmsgResponse(resp)
		break
	}

	return nil
}

func (mh *MessageHandler) processPrivmsgResponse(botResponse *server.BotResponse) {
	log.Printf("botResponse: %+v", botResponse)
	for _, value := range botResponse.Response {
		id := botResponse.Identifier
		_, ok := mh.Requests[id]
		if ok || id == mh.MagicIdentifier {
			mh.Connection.Privmsg(botResponse.Target, value)
			delete(mh.Requests, id)
		} else {
			log.Printf("Can't find an id entry for %+v\n", botResponse)
		}
	}
}

func (mh *MessageHandler) processActionResponse(botResponse *server.BotResponse) {

	for _, value := range botResponse.Response {
		id := botResponse.Identifier
		_, ok := mh.Requests[id]
		if ok || id == mh.MagicIdentifier {
			mh.Connection.Action(botResponse.Target, value)
			delete(mh.Requests, id)
		} else {
			log.Printf("Can't find an id entry for %+v\n", botResponse)
		}
	}
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
