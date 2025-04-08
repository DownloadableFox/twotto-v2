package commands

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

var _ api.Command = (*YiffCommand)(nil)

var DMPermission bool = true
var NSFW bool = true

type YiffCommand struct {
	service services.IE621Service
	logger  zerolog.Logger
}

func NewYiffCommand(service services.IE621Service, parent zerolog.Logger) *YiffCommand {
	return &YiffCommand{
		service: service,
		logger:  parent.With().Str("command", "yiff").Logger(),
	}
}

func (y *YiffCommand) Data() discordgo.ApplicationCommand {
	return discordgo.ApplicationCommand{
		Name:         "yiff",
		Description:  "Get a random yiff from e621",
		DMPermission: &DMPermission,
		NSFW:         &NSFW,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "random",
				Description: "Get a random post from e621",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "search",
				Description: "Search for posts on e621 based on tags",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "tags",
						Description: "Tags to search for",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "limit",
						Description: "Number of posts to return",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "page",
						Description: "Page to return",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "post",
				Description: "Get a specific post from e621",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "id",
						Description: "ID of the post to get",
						Required:    true,
					},
				},
			},
		},
	}
}

func (y *YiffCommand) Execute(c context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	data := i.ApplicationCommandData()

	// Defer the response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		return err
	}

	// Handle the subcommands
	switch data.Options[0].Name {
	case "random":
		return y.HandleRandom(c, s, i)
	case "search":
		return y.HandleSearch(c, s, i)
	case "post":
		return y.HandlePost(c, s, i)
	case "popular":
		return y.HandlePopular(c, s, i)
	}

	return nil
}

func (y *YiffCommand) GeneratePostEmbed(post *services.E621Post) *discordgo.MessageEmbed {
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

func (y *YiffCommand) PublishThread(s *discordgo.Session, channelID, messageID, tags string, posts []*services.E621Post) error {
	// Assume
	success := true

	// Create a thread to send the posts
	thr, err := s.MessageThreadStartComplex(channelID, messageID, &discordgo.ThreadStart{
		Name:                fmt.Sprintf("Posts with tags `%s`", tags),
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

func (y *YiffCommand) HandleRandom(ctx context.Context, s *discordgo.Session, e *discordgo.InteractionCreate) error {
	// Get the post
	post, err := y.service.GetRandomPost()
	if err != nil {
		return err
	}

	// Send the post
	embed := y.GeneratePostEmbed(post)

	// Create file request
	req, err := http.NewRequest(http.MethodGet, post.URL, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	file := &discordgo.File{
		Name:   fmt.Sprintf("post-%d.%s", post.ID, post.Ext),
		Reader: res.Body,
	}

	if _, err := s.InteractionResponseEdit(e.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
		Files:  []*discordgo.File{file},
	}); err != nil {
		return err
	}

	return nil
}

func (y *YiffCommand) HandleSearch(ctx context.Context, s *discordgo.Session, e *discordgo.InteractionCreate) error {
	data := e.ApplicationCommandData().Options[0]

	// Get tags
	tags, err := api.GetStringOption(data.Options, "tags")
	if err != nil {
		return err
	}

	// Get limit & page
	limit := api.GetIntegerDefaultOption(data.Options, "limit", 20)
	page := api.GetIntegerDefaultOption(data.Options, "page", 1)

	// Send the looking for posts embed
	startTime := time.Now()
	msg, err := s.InteractionResponseEdit(e.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{{
			Title:       "Looking for posts...",
			Description: "Searching for posts (this may take a while) ...",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Tags",
					Value:  fmt.Sprintf("`%s`", tags),
					Inline: true,
				},
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

	// Search for the posts
	posts, err := y.service.SearchPosts(tags, limit, page)
	if err != nil {
		return err
	}

	// If no posts were found
	if len(posts) == 0 {
		// Update interaction
		if _, err := s.InteractionResponseEdit(e.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{{
				Title:       "No posts found!",
				Description: "No posts were found with the given tags.\n**Note:**Some files may be too large to send (25MB limit).",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Tags",
						Value:  fmt.Sprintf("`%s`", tags),
						Inline: true,
					},
					{
						Name:   "Elapsed Time",
						Value:  fmt.Sprintf("`%s`", time.Since(startTime).Round(time.Second).String()),
						Inline: true,
					},
				},
				Color: api.ColorError,
			}},
		}); err != nil {
			return err
		}

		return nil
	}

	// Send the posts to a thread
	if err := y.PublishThread(s, msg.ChannelID, msg.ID, tags, posts); err != nil {
		// Operation cancelled
		s.InteractionResponseEdit(e.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{{
				Title:       "Operation cancelled! :(",
				Description: "There was an issue sending the posts to a thread.",
				Color:       api.ColorError,
			}},
		})

		return err
	}

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

	if _, err := s.InteractionResponseEdit(e.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{{
			Title:       "Posts sent!",
			Description: "Found posts have been sent below! **Note: **Some files might have been ommited as they were too large to send (25MB limit).",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Tags",
					Value:  fmt.Sprintf("`%s`", tags),
					Inline: false,
				},
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

func (y *YiffCommand) HandlePost(ctx context.Context, s *discordgo.Session, e *discordgo.InteractionCreate) error {
	data := e.ApplicationCommandData().Options[0]

	// Get the post
	postId, err := api.GetIntegerOption(data.Options, "id")
	if err != nil {
		return err
	}

	post, err := y.service.GetPostByID(postId)
	if err != nil {
		return err
	}

	// Send the post
	embed := y.GeneratePostEmbed(post)
	req, err := http.NewRequest(http.MethodGet, post.URL, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	file := &discordgo.File{
		Name:   fmt.Sprintf("post-%d.%s", post.ID, post.Ext),
		Reader: res.Body,
	}

	if _, err := s.InteractionResponseEdit(e.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
		Files:  []*discordgo.File{file},
	}); err != nil {
		return err
	}

	return nil
}

func (y *YiffCommand) HandlePopular(ctx context.Context, s *discordgo.Session, e *discordgo.InteractionCreate) error {
	// Get the post
	post, err := y.service.GetRandomPost()
	if err != nil {
		return err
	}

	// Send the post
	embed := y.GeneratePostEmbed(post)

	// Create file request
	req, err := http.NewRequest(http.MethodGet, post.URL, nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	file := &discordgo.File{
		Name:   fmt.Sprintf("post-%d.%s", post.ID, post.Ext),
		Reader: res.Body,
	}

	if _, err := s.InteractionResponseEdit(e.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
		Files:  []*discordgo.File{file},
	}); err != nil {
		return err
	}

	return nil
}
