package tasks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/DownloadableFox/twotto-v2/internal/api"
	"github.com/DownloadableFox/twotto-v2/internal/modules/yiff/services"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
)

var _ api.Task = (*PopularTask)(nil)

type PopularTask struct {
	logger  zerolog.Logger
	service services.IE621Service
}

func NewPopularTask(parent zerolog.Logger, service services.IE621Service) *PopularTask {
	return &PopularTask{
		logger:  parent.With().Str("task", "popular").Logger(),
		service: service,
	}
}

func (p *PopularTask) Data() api.TaskData {
	return api.TaskData{
		Name: "yiff-popular",
		// Cron: "48 13 * * *",
		Cron: "5 0 * * *",
		Once: false,
	}
}

func (p *PopularTask) Run(ctx context.Context, s *discordgo.Session) error {
	/*
		var channels = map[string]string{
			"1051532955056607302": "1051532956008718378",
		}

		// 1. Get all popular posts
		posts, err := p.service.GetPopularPosts()
		if err != nil {
			return err
		}

		if len(posts) == 0 {
			p.logger.Warn().Msg("No popular posts found- skipping day!")
			return nil
		}

		// 2. Send thread to selected servers
		wg := &sync.WaitGroup{}
		for _, guild := range s.State.Guilds {
			if channels[guild.ID] == "" {
				continue
			}

			// 3. Send information message
			wg.Add(1)
			go func() {
				defer wg.Done()

				if err := p.BeginThread(s, channels[guild.ID], posts); err != nil {
					p.logger.Error().Err(err).Msgf("Failed to begin thread for guild %s", guild.ID)
				}
			}()
		}

		// 4. Wait for all threads to finish
		wg.Wait()
		p.logger.Info().Msg("Finished sending popular posts to all servers!")
	*/

	p.logger.Info().Msg("test")
	return nil
}

func (p *PopularTask) BeginThread(s *discordgo.Session, channelID string, posts []*services.E621Post) error {
	// 1. Send the looking for posts embed
	startTime := time.Now()
	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       "Uploading popular posts...",
			Description: "Searching for posts (this may take a while) ...",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Elapsed Time",
					Value:  fmt.Sprintf("<t:%d:R>", startTime.Unix()),
					Inline: true,
				},
			},
			Color: api.ColorInfo,
		}},
	})
	if err != nil {
		return err
	}

	// 2. Start sending posts
	if err := p.PublishThread(s, channelID, msg.ID, posts); err != nil {
		// 2.5. Send error message
		s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:      msg.ID,
			Channel: channelID,

			Embeds: &[]*discordgo.MessageEmbed{{
				Title:       "Error sending posts!",
				Description: "There was an issue sending the posts to a thread.",
				Color:       api.ColorError,
			}},
		})

		return err
	}

	// 3. Update information message

	// Update interaction
	duration := time.Since(startTime)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	durationStr := ""
	if hours > 0 {
		durationStr += fmt.Sprintf("%dh ", hours)
	}

	if minutes > 0 {
		durationStr += fmt.Sprintf("%dm ", minutes)
	}

	if seconds > 0 {
		durationStr += fmt.Sprintf("%ds", seconds)
	}

	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:      msg.ID,
		Channel: channelID,

		Embeds: &[]*discordgo.MessageEmbed{{
			Title:       "Popular posts sent!",
			Description: "Found posts have been sent below! **Note: **Some files might have been ommited as they were too large to send (25MB limit).",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Count",
					Value:  fmt.Sprintf("%d", len(posts)),
					Inline: true,
				},
				{
					Name:   "Elapsed Time",
					Value:  durationStr,
					Inline: true,
				},
			},
			Color: api.ColorSuccess,
		}},
	}); err != nil {
		return err
	}

	return nil
}

func (y *PopularTask) PublishThread(s *discordgo.Session, channelID, messageID string, posts []*services.E621Post) error {
	// Assume
	success := true

	// Create a thread to send the posts
	thr, err := s.MessageThreadStartComplex(channelID, messageID, &discordgo.ThreadStart{
		Name:                fmt.Sprintf("Popular posts of the day (%s)", time.Now().Format("2006-01-02")),
		AutoArchiveDuration: 60, // 1 hour
		Invitable:           true,
	})
	if err != nil {
		return err
	}

	// Send the posts
	for _, post := range posts {
		s.ChannelTyping(thr.ID)

		embed := y.GeneratePostEmbed(post)
		req, err := http.NewRequest(http.MethodGet, post.URL, nil)
		if err != nil {
			y.logger.Warn().Err(err).Msgf("Failed to create request for post #%d (source: %s)", post.ID, post.URL)
			success = false
			continue
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			y.logger.Warn().Err(err).Msgf("Failed to download post #%d (source: %s)", post.ID, post.URL)
			success = false
			continue
		}
		defer res.Body.Close()

		file := &discordgo.File{
			Name:   fmt.Sprintf("post-%d.%s", post.ID, post.Ext),
			Reader: res.Body,
		}

		if _, err := s.ChannelMessageSendComplex(thr.ID, &discordgo.MessageSend{
			Embed: embed,
			Files: []*discordgo.File{file},
		}); err != nil {
			y.logger.Warn().Err(err).Msgf("Failed to send post #%d (source: %s)", post.ID, post.URL)
			success = false
			continue
		}
	}

	if !success {
		s.ChannelMessageSendComplex(thr.ID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       "Failed to send some posts!",
				Description: "There was an issue sending some posts. These posts were omitted from the thread!",
				Color:       api.ColorWarning,
			}},
		})
	}

	return nil
}

func (y *PopularTask) GeneratePostEmbed(post *services.E621Post) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("E621 Post #%d", post.ID),
		Description: "You can find this post by clicking on the following URL.",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Extension",
				Value:  fmt.Sprintf("`%s`", post.Ext),
				Inline: true,
			},
			{
				Name:   "Size",
				Value:  fmt.Sprintf("%.2f MB", float64(post.Size)/1024.0/1024.0),
				Inline: true,
			},
		},
		URL:   fmt.Sprintf("https://e621.net/posts/%d", post.ID),
		Color: api.ColorInfo,
	}

	return embed
}
