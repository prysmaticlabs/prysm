package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Variables used for command line parameters
var (
	Token         string
	APIUrl        string
	RPCUrl        string
	EncryptedPriv string
	Password      string
	DBPath        string
	DenylistPath  string
	Debug         bool

	conn         *grpc.ClientConn
	beaconClient eth.BeaconChainClient
	nodeClient   eth.NodeClient

	log = logrus.WithField("prefix", "prysmBot")
)

func init() {
	flag.StringVar(&Token, "token", "", "Bot Token")
	flag.StringVar(&APIUrl, "api-url", "", "API Url for gRPC")
	flag.StringVar(&RPCUrl, "rpc-url", "", "RPC Url for Goerli network")
	flag.StringVar(&EncryptedPriv, "private-key", "", "Private key for Goerli wallet")
	flag.StringVar(&Password, "password", "", "Password for encrypted private key")
	flag.StringVar(&DenylistPath, "denylist", "", "Filepath to denylist of regular expressions")
	flag.BoolVar(&Debug, "debug", false, "Enable debug logging")
	flag.Parse()

	if Debug {
		logrus.SetLevel(logrus.DebugLevel)
		log.Debug("Debug logging enabled.")
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	conn, err = grpc.Dial(APIUrl, grpc.WithInsecure())
	if err != nil {
		log.Errorf("Failed to dial: %v", err)
		return
	}
	beaconClient = eth.NewBeaconChainClient(conn)
	nodeClient = eth.NewNodeClient(conn)
	defer func() {
		if err := conn.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if err := initWallet(); err != nil {
		log.Error(err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(messageReaction)

	// Monitor denylist changes
	go monitorDenylistFile(ctx, DenylistPath)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	defer func() {
		if err := dg.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	if deniedMessage(s, m) {
		return
	}
	if !whitelistedChannel(m.ChannelID) {
		return
	}
	// Ignore all messages that don't start with "!".
	if !strings.HasPrefix(m.Content, "!") {
		return
	}
	if err := s.ChannelTyping(m.ChannelID); err != nil {
		log.WithError(err).Error("Cannot send typing notification to channel")
	}

	fullCommand := m.Content[1:]
	// If the message is "ping" reply with "Pong!"
	if fullCommand == "ping" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Pong!")
		if err != nil {
			log.WithError(err).Errorf("Error sending embed %s", fullCommand)
		}
		return
	}
	if fullCommand == "help" && helpOkayChannel(m.ChannelID) {
		embed := fullHelpEmbed()
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			log.WithError(err).Errorf("Error sending embed %s", fullCommand)
		}
		return
	}

	if isRandomCommand(fullCommand) {
		result := getRandomResult(fullCommand)
		_, err := s.ChannelMessageSend(m.ChannelID, result)
		if err != nil {
			log.WithError(err).Errorf("Error handling command %s", fullCommand)
			return
		}
	}

	splitCommand := strings.Split(fullCommand, ".")
	if fullCommand == splitCommand[0] {
		splitCommand = strings.Split(fullCommand, " ")
		if splitCommand[0] == "send" && goerliOkayChannel(m.ChannelID) {
			if err := validateUser(m); err != nil {
				log.WithError(err).Error("Failed to validate user")
				if _, err2 := s.ChannelMessageSend(m.ChannelID, err.Error()); err2 != nil {
					log.WithError(err2).Error("Could not send message")
				}
				return
			}
			resp, err := SendGoeth(splitCommand[1:])
			if err != nil {
				log.WithError(err).Error("Could not send goerli eth")
				return
			}
			_, err = s.ChannelMessageSend(m.ChannelID, resp)
			if err != nil {
				log.WithError(err).Errorf("Error handling command %s", fullCommand)
			}
		}
		return
	}

	if len(splitCommand) > 1 && strings.TrimSpace(splitCommand[1]) == "" {
		return
	}
	commandGroup := splitCommand[0]
	endOfCommand := strings.Index(splitCommand[1], " ")
	var parameters []string
	if endOfCommand == -1 {
		endOfCommand = len(splitCommand[1])
	} else {
		parameters = strings.Split(splitCommand[1][endOfCommand:], ",")
		for i, param := range parameters {
			parameters[i] = strings.TrimSpace(param)
		}
	}
	command := splitCommand[1][:endOfCommand]

	var cmdFound bool
	var cmdGroupFound bool
	var reqGroup *botCommandGroup
	for _, flagGroup := range allFlagGroups {
		if flagGroup.name == commandGroup || flagGroup.shorthand == commandGroup {
			cmdGroupFound = true
			reqGroup = flagGroup
			for _, cmd := range reqGroup.commands {
				if command == cmd.command || command == cmd.shorthand || command == "help" {
					cmdFound = true
				}
			}
		}
	}
	if !cmdGroupFound || !cmdFound {
		return
	}

	if command == "help" && helpOkayChannel(m.ChannelID) {
		embed := specificHelpEmbed(reqGroup)
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			log.WithError(err).Errorf("Error sending embed %s", fullCommand)
		}
		return
	}

	var result string
	switch commandGroup {
	case currentCommandGroup.name, currentCommandGroup.shorthand:
		result = getHeadCommandResult(command)
	case stateCommandGroup.name, stateCommandGroup.shorthand:
		result = getStateCommandResult(command, parameters)
	case valCommandGroup.name, valCommandGroup.shorthand:
		result = getValidatorCommandResult(command, parameters)
	case blockCommandGroup.name, blockCommandGroup.shorthand:
		result = getBlockCommandResult(command, parameters)
	default:
		result = "Command not found, sorry!"
	}
	if result == "" {
		return
	}
	_, err := s.ChannelMessageSend(m.ChannelID, result)
	if err != nil {
		log.WithError(err).Errorf("Error handling command %s", fullCommand)
		return
	}
}

func helpOkayChannel(channelID string) bool {
	switch channelID {
	case prysmInternal:
		return true
	case personalTesting:
		return true
	case prysmRandom:
		return true
	default:
		return false
	}
}

func goerliOkayChannel(channelID string) bool {
	switch channelID {
	case personalTesting:
		return true
	case prysmGoerli:
		return true
	default:
		return false
	}
}

func whitelistedChannel(channelID string) bool {
	switch channelID {
	case prysmGeneral:
		return true
	case prysmGoerli:
		return true
	default:
		return helpOkayChannel(channelID)
	}
}

func messageReaction(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	// Ignore reactions by the bot.
	if m.UserID == s.State.User.ID {
		return
	}
	handleDenyListMessageReaction(s, m)
}
