package txt

import (
	"regexp"
	"strings"
	"time"
)

type Chronological struct {
	Time time.Time `json:"time" title:"Timestamp" description:"In RFC 3339 format, e.g. 2020-01-01T01:02:03Z"`
	Text string    `json:"text" title:"Text" formType:"textarea" description:"Text, can contain HTML."`
}

type Replace struct {
	re *regexp.Regexp

	IsRegex bool   `json:"isRegex,omitempty" noTitle:"true" inlineTitle:"Regular expression"`
	From    string `json:"from" title:"From" formType:"textarea"`
	To      string `json:"to" title:"To" formType:"textarea"`
}

type Replaces []Replace

func (rr Replaces) Apply(o *RenderOptions) {
	o.Replaces = rr
}

func (rr Replaces) Replace(s string) (_ string, err error) {
	for i, r := range rr {
		if r.IsRegex {
			if r.re == nil {
				r.re, err = regexp.Compile(r.From)
				if err != nil {
					return "", err
				}

				rr[i] = r
			}

			s = r.re.ReplaceAllString(s, r.To)
		} else {
			s = strings.ReplaceAll(s, r.From, r.To)
		}
	}

	return s, nil
}
