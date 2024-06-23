package main

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/stretchr/testify/assert"
)

const (
	ValidTwoLinks = `<!-- chainlink -->
- [ ] #1
- [ ] #2
`
)

var (
	TestIssue = ChainIssue{
		Repo: repository.Repository{
			Host:  "github.com",
			Name:  "gh-chainlink",
			Owner: "RoryQ",
		},
		Number: 1,
		Title:  "Issue 1",
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
			content: ValidTwoLinks,
			want: &Chain{
				Items: []ChainItem{
					{
						ChainIssue: ChainIssue{},
						Message:    "#1",
						Checked:    false,
						Raw:        "",
					},
				},
				Raw: "",
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
