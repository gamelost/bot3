package handlers

// this file contains the basic set of event handlers
// to manage tracking an irc connection etc.

import (
	"encoding/json"
	nsq "github.com/bitly/go-nsq"
	irc "github.com/fluffle/goirc/client"
	"github.com/gamelost/bot3server/server"
)

func H_LINERPC(line *irc.Line, writer *nsq.Writer) {

	// create request payload
}
