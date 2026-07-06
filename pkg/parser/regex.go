package parser

import (
	"regexp"
	"strconv"
	"strings"
)

type NormalizedTvMedia struct {
	BaseTitle     string
	SeasonNumber  int
	EpisodeNumber int
}

var (
	// sxxExxRegex matches S01E04 preceded by spaces, dots, underscores, or dashes.
	sxxExxRegex = regexp.MustCompile(`(?i)[\s._-]+S(\d{1,2})E(\d{1,3})`)

	// spliceRegex matches 1x02 preceded by spaces, dots, underscores, or dashes.
	spliceRegex = regexp.MustCompile(`(?i)[\s._-]+(\d{1,2})x(\d{1,3})`)

	// verboseRegex matches plain-text Season/Episode notation.
	verboseRegex = regexp.MustCompile(`(?i)[\s._-]+Season\s+(\d{1,2})[\s._-]+Episode\s+(\d{1,3})`)

	// junkRegex mandates strict word boundaries (\b) so it won't clip letters inside real words.
	junkRegex = regexp.MustCompile(`(?i)\b(?:\d{3,4}p|bluray|hdtv|x26[45]|web-dl|amzn|nf)\b.*`)
)

func NormalizeTvEntry(rawTitle string) NormalizedTvMedia {
	cleaned := rawTitle
	season := 1
	episode := 1
	matched := false

	if loc := sxxExxRegex.FindStringSubmatchIndex(cleaned); loc != nil {
		season, _ = strconv.Atoi(cleaned[loc[2]:loc[3]])
		episode, _ = strconv.Atoi(cleaned[loc[4]:loc[5]])
		cleaned = cleaned[:loc[0]]
		matched = true
	}

	if !matched {
		if loc := spliceRegex.FindStringSubmatchIndex(cleaned); loc != nil {
			season, _ = strconv.Atoi(cleaned[loc[2]:loc[3]])
			episode, _ = strconv.Atoi(cleaned[loc[4]:loc[5]])
			cleaned = cleaned[:loc[0]]
			matched = true
		}
	}

	if !matched {
		if loc := verboseRegex.FindStringSubmatchIndex(cleaned); loc != nil {
			season, _ = strconv.Atoi(cleaned[loc[2]:loc[3]])
			episode, _ = strconv.Atoi(cleaned[loc[4]:loc[5]])
			cleaned = cleaned[:loc[0]]
		}
	}

	// Clean up formatting delimiters before doing junk parsing or trimming
	cleaned = strings.ReplaceAll(cleaned, ".", " ")
	cleaned = strings.ReplaceAll(cleaned, "_", " ")

	cleaned = junkRegex.ReplaceAllString(cleaned, "")

	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimFunc(cleaned, func(r rune) bool {
		return r == '-' || r == ':' || r == ','
	})
	cleaned = strings.TrimSpace(cleaned)

	return NormalizedTvMedia{
		BaseTitle:     cleaned,
		SeasonNumber:  season,
		EpisodeNumber: episode,
	}
}
