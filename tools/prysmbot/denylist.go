package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fsnotify/fsnotify"
)

var denylist []*regexp.Regexp

func monitorDenylistFile(ctx context.Context, fp string) {
	log.WithField("filepath", fp).Info("Monitoring denylist for file changes")
	updateDenyList(fp)

	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Error("Failed to create fsnotify watcher")
		return
	}
	if err := w.Add(fp); err != nil {
		log.WithError(err).Error("Failed to create fsnotify watcher")
		return
	}
	for {
		select {
		case <-w.Events:
			updateDenyList(fp)
		case <-ctx.Done():
			return
		}
	}
}

func updateDenyList(fp string) {
	newDenyList := make([]*regexp.Regexp, 0)
	content, err := ioutil.ReadFile(fp)
	if err != nil {
		log.WithError(err).Error("Failed to read denylist")
		return
	}
	s := string(content)
	for _, row := range strings.Split(s, "\n") {
		if row == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + row) // Prefix (?i) to make case insenstive.
		if err != nil {
			log.WithError(err).Errorf("Failed to parse regex: %s", row)
			continue
		}
		newDenyList = append(newDenyList, re)
	}

	if len(newDenyList) > 0 {
		denylist = newDenyList
		log.WithField("count", len(newDenyList)).Info("Updated deny list")
	}
}

func deniedMessage(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	for _, re := range denylist {
		if re.MatchString(m.Content) {
			deleteMessage(s, m)
			handleDenyListMessage(s, m, re)
			return true
		}
	}

	return false
}

func deleteMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		log.WithError(err).Error("Failed to delete denied message.")
	}
}

func handleDenyListMessage(s *discordgo.Session, m *discordgo.MessageCreate, re *regexp.Regexp) {
	ts, err := discordgo.SnowflakeTimestamp(m.Author.ID)
	if err != nil {
		log.WithError(err).Error("Could not determine user's timestamp")
	}
	age := time.Since(ts)
	message := fmt.Sprintf("User %s (ID:%s) sent the following content:\n"+
		"> %s\n"+
		"Their message matched denylist filter %s and was deleted.\n"+
		"%s's account age is: %v\n"+
		"React with :hammer: to auto-ban.",
		m.Author.Username,
		m.Author.ID,
		m.Content,
		re.String(),
		m.Author.Username,
		age)

	if _, err := s.ChannelMessageSend(prysmInternal, message); err != nil {
		log.WithError(err).Error("Failed to notify prysm internal channel of denied message.")
		return
	}
	if err := s.MessageReactionAdd(m.ChannelID, m.ID, "🔨"); err != nil {
		log.WithError(err).Error("Failed to react to message")
	}
}

func handleDenyListMessageReaction(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.ChannelID != prysmInternal {
		return
	}
	if m.Emoji.Name != "🔨" {
		log.WithField("emoji", m.Emoji.Name).Debug("Invalid emoji reaction")
		return
	}
	om, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		log.WithError(err).Error("Failed to get original message with reaction")
		return
	}
	// Ignore all messages not created by the bot itself.
	if om.Author.ID != s.State.User.ID {
		return
	}

	re := regexp.MustCompile(`\(ID:(\\d*)\)`)
	matches := re.FindStringSubmatch(om.Content)
	if len(matches) < 2 {
		log.Errorf("Could not extract user ID from message: %s", om.Content)
		return
	}
	userID := matches[1]

	log.WithField("userID", userID).Debug("Banning user")
	if err := s.GuildBanCreateWithReason(m.GuildID, userID, "Posting forbidden content", 0 /*days*/); err != nil {
		log.WithError(err).Error("Failed to ban user")
		return
	}
	if _, err := s.ChannelMessageSend(prysmInternal, fmt.Sprintf("Banned user %s", userID)); err != nil {
		log.WithError(err).Error("Failed to send message")
	}
}
