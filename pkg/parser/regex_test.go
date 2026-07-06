package parser

import (
	"testing"
)

func TestNormalizeTvEntry(t *testing.T) {
	// Define the testing matrix mapping raw inputs to exact expected data structures
	tests := []struct {
		name     string
		input    string
		wantName string
		wantS    int
		wantE    int
	}{
		{
			name:     "Standard SxxExx Notation with dots",
			input:    "The.Sopranos.S01E04.1080p.BluRay.x264",
			wantName: "The Sopranos",
			wantS:    1,
			wantE:    4,
		},
		{
			name:     "Classic Splice Notation with dashes",
			input:    "Breaking Bad - 1x02 - Cat's in the Bag",
			wantName: "Breaking Bad",
			wantS:    1,
			wantE:    2,
		},
		{
			name:     "Verbose Prose Notation with spacing",
			input:    "The Office (US) Season 3 Episode 10",
			wantName: "The Office (US)",
			wantS:    3,
			wantE:    10,
		},
		{
			name:     "Lowercase s/e notation with trailing junk",
			input:    "Stranger.Things.s04e08.web-dl.nf.x265",
			wantName: "Stranger Things",
			wantS:    4,
			wantE:    8,
		},
		{
			name:     "Fallback resolution on unmatched string patterns",
			input:    "Some Random Unformatted TV Show Name",
			wantName: "Some Random Unformatted TV Show Name",
			wantS:    1,
			wantE:    1,
		},
	}

	// Loop over each structural row in the table
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTvEntry(tt.input)

			if got.BaseTitle != tt.wantName {
				t.Errorf("NormalizeTvEntry() BaseTitle = %q, want %q", got.BaseTitle, tt.wantName)
			}
			if got.SeasonNumber != tt.wantS {
				t.Errorf("NormalizeTvEntry() SeasonNumber = %d, want %d", got.SeasonNumber, tt.wantS)
			}
			if got.EpisodeNumber != tt.wantE {
				t.Errorf("NormalizeTvEntry() EpisodeNumber = %d, want %d", got.EpisodeNumber, tt.wantE)
			}
		})
	}
}
