package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/cometline/cometmind/internal/jobs"
)

// DiscordJobNotifier delivers job lifecycle messages to Discord channels.
type DiscordJobNotifier struct {
	Reply func(context.Context, OutboundMessage) error
}

func (n DiscordJobNotifier) OnJobEvent(ctx context.Context, job jobs.Job, action, detail string) {
	if n.Reply == nil || job.SourceChannelID == "" {
		return
	}
	var text string
	switch action {
	case jobs.EventClaimed:
		text = fmt.Sprintf("Job %s claimed.", job.ID)
	case jobs.EventCompleted:
		text = fmt.Sprintf("Job %s completed.", job.ID)
		if strings.TrimSpace(detail) != "" {
			text += "\n" + detail
		}
	case jobs.EventReleased, jobs.EventLeaseExpired:
		text = fmt.Sprintf("Job %s released (%s).", job.ID, detail)
	default:
		return
	}
	_ = n.Reply(ctx, OutboundMessage{
		Platform:  "discord",
		ChannelID: job.SourceChannelID,
		Text:      text,
	})
}
