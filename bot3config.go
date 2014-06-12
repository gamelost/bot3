package main

import (
	iniconf "code.google.com/p/goconf/conf"
	"strings"
)

// // set up the config struct
// botNick, _ := b.Config.GetString(CONFIG_CAT_DEFAULT, "nick")
// botPass, _ := b.Config.GetString(CONFIG_CAT_DEFAULT, "pass")
// botServer, _ := b.Config.GetString(CONFIG_CAT_DEFAULT, "ircserver")
// chanToJoin, _ := b.Config.GetString(CONFIG_CAT_DEFAULT, "channel")
// bot3serverOutput, _ := b.Config.GetString(CONFIG_CAT_DEFAULT, "bot3server-output")
// bot3serverInput, _ := b.Config.GetString(CONFIG_CAT_DEFAULT, "bot3server-input")

const (
	// config categories
	CONFIG_CAT_DEFAULT = "default"
	CONFIG_CAT_NSQ     = "nsq"
	CONFIG_CAT_BOT     = "bot"
)

type Bot3Config struct {
	Config           *iniconf.ConfigFile
	BotNick          string
	BotPass          string
	BotIRCServer     string
	BotChannelToJoin string
	BotAdminNicks    []string
	BotOfflinePrefix string
	// nsq topics
	Bot3ServerOutputTopic string
	Bot3ServerInputTopic  string
	MagicIdentifier       string
}

func Bot3ConfigFromConfigFile(configFile *iniconf.ConfigFile) (*Bot3Config, error) {

	conf := &Bot3Config{Config: configFile}

	// read in required properties
	err := conf.readInRequiredProperties()
	if err != nil {
		return nil, err
	}

	// read in other optional properties

	return conf, nil
}

func (bc *Bot3Config) readInRequiredProperties() error {

	bc.BotNick, _ = bc.Config.GetString(CONFIG_CAT_BOT, "nick")
	bc.BotPass, _ = bc.Config.GetString(CONFIG_CAT_BOT, "pass")
	bc.BotIRCServer, _ = bc.Config.GetString(CONFIG_CAT_DEFAULT, "ircserver")
	bc.BotChannelToJoin, _ = bc.Config.GetString(CONFIG_CAT_DEFAULT, "channel")
	bc.Bot3ServerOutputTopic, _ = bc.Config.GetString(CONFIG_CAT_NSQ, "bot3server-output")
	bc.Bot3ServerInputTopic, _ = bc.Config.GetString(CONFIG_CAT_NSQ, "bot3server-input")
	bc.MagicIdentifier, _ = bc.Config.GetString(CONFIG_CAT_NSQ, "magic-identifier")
	bc.BotOfflinePrefix, _ = bc.Config.GetString(CONFIG_CAT_BOT, "offlineNickPrefix")

	csv, _ := bc.Config.GetString(CONFIG_CAT_BOT, "adminNicks")
	bc.BotAdminNicks = strings.Split(csv, ",")

	return nil
}

func (bc *Bot3Config) IsAdminNick(nick string) bool {

	for _, n := range bc.BotAdminNicks {
		if n == nick {
			return true
		}
	}

	return false
}
