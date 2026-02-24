package usecase

import (
	"html/template"

	"github.com/vearutop/photo-blog/pkg/txt"
)

type albumTimelineItem struct {
	Image *Image
	Text  template.HTML
	Ts    int64
}

func buildAlbumTimeline(images []Image, texts []txt.Chronological, newestFirst bool) []albumTimelineItem {
	if len(images) == 0 && len(texts) == 0 {
		return nil
	}

	remaining := make([]txt.Chronological, len(texts))
	copy(remaining, texts)

	timeline := make([]albumTimelineItem, 0, len(images)+len(texts))

	for _, img := range images {
		if img.Is360Pano {
			continue
		}

		if len(remaining) > 0 {
			next := remaining[:0]
			for _, t := range remaining {
				tt := t.Time.Unix()
				if newestFirst {
					if tt < img.UTime {
						next = append(next, t)
						continue
					}
				} else {
					if tt > img.UTime {
						next = append(next, t)
						continue
					}
				}

				timeline = append(timeline, albumTimelineItem{
					Text: template.HTML(t.Text),
					Ts:   tt,
				})
			}

			remaining = next
		}

		imgCopy := img
		timeline = append(timeline, albumTimelineItem{
			Image: &imgCopy,
			Ts:    img.UTime,
		})
	}

	for _, t := range remaining {
		timeline = append(timeline, albumTimelineItem{
			Text: template.HTML(t.Text),
			Ts:   t.Time.Unix(),
		})
	}

	return timeline
}
