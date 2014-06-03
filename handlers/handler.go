package handlers

// this file contains the basic set of event handlers
// to manage tracking an irc connection etc.

import (
	"encoding/json"
	nsq "github.com/gamelost/go-nsq"
	"github.com/gamelost/bot3server/server"
	irc "github.com/gamelost/goirc/client"
)

func H_LINERPC(line *irc.Line, writer *nsq.Writer) {

	// create request payload
}
