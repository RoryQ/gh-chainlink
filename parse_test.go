package main

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/stretchr/testify/assert"
)

const (
	ValidTwoLinks = `<!-- chainlink -->
- [ ] #1
- [x] #2 &larr; This PR`
	NumberedItems = `<!-- chainlink -->
1. #1
2. #2 &larr; This PR`
	BulletedItems = `<!-- chainlink -->
- #1
- #2 &larr; This PR`
)

var (
	TestIssue = ChainIssue{
		Repo: repository.Repository{
			Host:  "github.com",
			Name:  "gh-chainlink",
			Owner: "RoryQ",
		},
		Number: 1,
	}
)

func TestParse(t *testing.T) {
	tests := map[string]struct {
		current   ChainIssue
		content   string
		want      *Chain
		errAssert assert.ErrorAssertionFunc
	}{
		"ValidTwoLinks": {
			current: TestIssue,
			content: ValidTwoLinks,
			want: &Chain{
				Source:  TestIssue,
				Current: TestIssue,
				Items: []ChainItem{
					{
						ChainIssue: ChainIssue{
							Repo:   TestIssue.Repo,
							Number: 1,
						},
						IsCurrent: true,
						Message:   "#1",
						ItemState: Unchecked,
						Raw:       "- [ ] #1",
					},
					{
						ChainIssue: ChainIssue{
							Repo:   TestIssue.Repo,
							Number: 2,
						},
						IsCurrent: false,
						Message:   "#2",
						ItemState: Checked,
						Raw:       "- [x] #2 &larr; This PR",
					},
				},
				Raw: "- [ ] #1\n- [x] #2 &larr; This PR",
			},
			errAssert: assert.NoError,
		},
		"NumberedItems": {
			current: TestIssue,
			content: NumberedItems,
			want: &Chain{
				Source:  TestIssue,
				Current: TestIssue,
				Items: []ChainItem{
					{
						ChainIssue: ChainIssue{
							Repo:   TestIssue.Repo,
							Number: 1,
						},
						IsCurrent: true,
						Message:   "#1",
						ItemState: Unchecked,
						Raw:       "1. #1",
					},
					{
						ChainIssue: ChainIssue{
							Repo:   TestIssue.Repo,
							Number: 2,
						},
						IsCurrent: false,
						Message:   "#2",
						ItemState: Unchecked,
						Raw:       "2. #2 &larr; This PR",
					},
				},
				Raw: "1. #1\n2. #2 &larr; This PR",
			},
			errAssert: assert.NoError,
		},
		"BulletedItems": {
			current: TestIssue,
			content: BulletedItems,
			want: &Chain{
				Source:  TestIssue,
				Current: TestIssue,
				Items: []ChainItem{
					{
						ChainIssue: ChainIssue{
							Repo:   TestIssue.Repo,
							Number: 1,
						},
						IsCurrent: true,
						Message:   "#1",
						ItemState: Unchecked,
						Raw:       "- #1",
					},
					{
						ChainIssue: ChainIssue{
							Repo:   TestIssue.Repo,
							Number: 2,
						},
						IsCurrent: false,
						Message:   "#2",
						ItemState: Unchecked,
						Raw:       "- #2 &larr; This PR",
					},
				},
				Raw: "- #1\n- #2 &larr; This PR",
			},
			errAssert: assert.NoError,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(tt.current, tt.content)
			tt.errAssert(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_issueFromMessage(t *testing.T) {
	tests := map[string]struct {
		current ChainIssue
		message string
		want    ChainIssue
	}{
		"NumberShorthand": {
			current: TestIssue,
			message: "#123",
			want: ChainIssue{
				Repo:   TestIssue.Repo,
				Number: 123,
			},
		},
		"GithubLink": {
			current: TestIssue,
			message: "https://github.com/owner/repo/issues/123",
			want: ChainIssue{
				Repo: repository.Repository{
					Host:  "github.com",
					Name:  "repo",
					Owner: "owner",
				},
				Number: 123,
			},
		},
		"GithubEnterpriseLink": {
			current: TestIssue,
			message: "https://github.enterprise.com/owner/repo/issues/123",
			want: ChainIssue{
				Repo: repository.Repository{
					Host:  "github.enterprise.com",
					Name:  "repo",
					Owner: "owner",
				},
				Number: 123,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, issueFromMessage(tt.current, tt.message))
		})
	}
}

func TestReplaceChain(t *testing.T) {
	tests := map[string]struct {
		body  string
		chain string
		want  string
	}{
		"BodyHasChainlink": {
			body:  "<!--chainlink-->\n\n1. #1",
			chain: "<!--chainlink-->\n1. #1 &larr; This PR",
			want:  "<!--chainlink-->\n1. #1 &larr; This PR",
		},
		"BodyHasIndicatorOnly": {
			body:  "<!--chainlink-->\n\nSome Text.",
			chain: "<!--chainlink-->\n1. #1 &larr; This PR",
			want:  "<!--chainlink-->\n1. #1 &larr; This PR\n\nSome Text.",
		},
		"NoIndicator": {
			body:  "Some Text.",
			chain: "<!--chainlink-->\n1. #1 &larr; This PR",
			want:  "Some Text.\n<!--chainlink-->\n1. #1 &larr; This PR",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ReplaceChain(tt.body, tt.chain), "ReplaceChain(%v, %v)", tt.body, tt.chain)
		})
	}
}
