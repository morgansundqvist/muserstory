package domain

import (
	"testing"
)

func TestParseMarkdownFileContent(t *testing.T) {

	simpleMarkdown := `
- As a user, I want to be able to log in so that I can access my account.
- As a user, I want to be able to log out so that I can secure my account.
`

	complexMarkdown := `
---
title: "Complex Markdown"
author: "John Doe"
date: "2023-10-01"
---
# Summary
Super simple app

** Category: **
- As a user, I want to be able to log in so that I can access my account.
- As a user, I want to be able to log out so that I can secure my account.
- As a user, I want to be able to reset my password so that I can regain access to my account.
`

	type args struct {
		content string
	}
	tests := []struct {
		name            string
		args            args
		want            *MarkdownFile
		wantErr         bool
		numberOfStories int
	}{
		{
			name: "simple markdown",
			args: args{
				content: simpleMarkdown,
			},
			want: &MarkdownFile{
				Metadata: map[string]interface{}{},
				Summary:  "",
				Stories: []UserStory{
					{
						Description: "As a user, I want to be able to log in so that I can access my account.",
					},
					{
						Description: "As a user, I want to be able to log out so that I can secure my account.",
					},
				},
			},
			wantErr:         false,
			numberOfStories: 2,
		},

		{
			name: "complex markdown",
			args: args{
				content: complexMarkdown,
			},
			want: &MarkdownFile{
				Metadata: map[string]interface{}{}, //Todo add metadata
				Summary:  "Super simple app",
				Stories: []UserStory{
					{
						Description: "As a user, I want to be able to log in so that I can access my account.",
					},
					{
						Description: "As a user, I want to be able to log out so that I can secure my account.",
					},
					{
						Description: "As a user, I want to be able to reset my password so that I can regain access to my account.",
					},
				},
			},
			wantErr:         false,
			numberOfStories: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMarkdownFileContent(tt.args.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMarkdownFileContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil || tt.want == nil {
				if got != tt.want {
					t.Errorf("ParseMarkdownFileContent() = %v, want %v", got, tt.want)
				}
				return
			}
			if tt.numberOfStories != len(tt.want.Stories) {
				t.Errorf("ParseMarkdownFileContent() number of stories = %v, want %v", len(tt.want.Stories), tt.numberOfStories)
			}
			if got.Summary != tt.want.Summary {
				t.Errorf("ParseMarkdownFileContent() Summary = %v, want %v", got.Summary, tt.want.Summary)
			}
		})
	}
}
