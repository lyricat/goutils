package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

type (
	Config struct {
		BotClientID string
		Token       string
	}
	Client struct {
		cfg       Config
		me        *discordgo.User
		dgSession *discordgo.Session
	}
)

func New(cfg Config) *Client {
	dgSession, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		slog.Error("[goutils.discord] error creating Discord session", "error", err)
		return nil
	}
	me, err := dgSession.User("@me")
	if err != nil {
		slog.Error("[goutils.discord] error getting myself", "error", err)
		return nil
	}
	return &Client{
		cfg:       cfg,
		me:        me,
		dgSession: dgSession,
	}
}

func (b *Client) SendTextMessage(channelID, title, summary, url string) error {
	content := fmt.Sprintf("# %s\n\n%s\n\n👉 %s\n\n", title, summary, url)
	if _, err := b.dgSession.ChannelMessageSend(channelID, content); err != nil {
		slog.Error("[goutils.discord] error sending message", "error", err)
		return err
	}

	return nil
}

func (b *Client) CheckPermission(channelID string) (bool, error) {
	channel, err := b.dgSession.Channel(channelID)
	if err != nil {
		return false, err
	}

	guild, err := b.dgSession.Guild(channel.GuildID)
	if err != nil {
		return false, err
	}

	member, err := b.dgSession.GuildMember(guild.ID, b.me.ID)
	if err != nil {
		return false, err
	}

	for _, roleID := range member.Roles {
		for _, guildRole := range guild.Roles {
			if guildRole.ID == roleID {
				if guildRole.Permissions&discordgo.PermissionSendMessages != 0 {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
