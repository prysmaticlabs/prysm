package main

import (
	"errors"
	"time"

	"github.com/bwmarrin/discordgo"
)

var twoWeeks = time.Hour * 24 * 14

func validateUser(m *discordgo.MessageCreate) error {
	if m == nil || m.Author == nil {
		log.Error("nil message or message author")
		return errors.New("internal error")
	}
	t, err := discordgo.SnowflakeTimestamp(m.Author.ID)
	if err != nil {
		return err
	}
	if t.After(time.Now().Add(-1 * twoWeeks)) {
		log.WithField("creation_date", t).Debug("User created too recently")
		return errors.New("account created too recently")
	}
	return nil
}
