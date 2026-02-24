package usecase

import (
	"strings"

	"github.com/vearutop/photo-blog/internal/domain/photo"
)

func buildAISays(meta *photo.MetaData) string {
	if meta == nil {
		return ""
	}

	text := ""

	for _, d := range meta.ImageDescriptions {
		if strings.TrimSpace(d.Text) == "" {
			continue
		}

		text = d.Text
		break
	}

	if text == "" {
		for _, l := range meta.ImageClassification {
			if l.Model != "cf-uform-gen2" {
				continue
			}

			if strings.TrimSpace(l.Text) != "" {
				text = l.Text

				break
			}
		}
	}

	return text
}
