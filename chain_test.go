package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainItem_RenderMarkdown(t *testing.T) {
	type fields struct {
		ChainIssue ChainIssue
		IsCurrent  bool
		Message    string
		ItemState  ItemState
		Raw        string
	}
	tests := map[string]struct {
		fields fields
		want   string
	}{
		"Checked": {
			fields: fields{
				ChainIssue: TestIssue,
				Message:    "#1234",
				ItemState:  Checked,
			},
			want: "- [x] #1234",
		},
		"Current": {
			fields: fields{
				ChainIssue: TestIssue,
				Message:    "#123",
				IsCurrent:  true,
			},
			want: "- [ ] #123 " + CurrentIndicator,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i := ChainItem{
				ChainIssue: tt.fields.ChainIssue,
				IsCurrent:  tt.fields.IsCurrent,
				Message:    tt.fields.Message,
				ItemState:  tt.fields.ItemState,
				Raw:        tt.fields.Raw,
			}
			assert.Equalf(t, tt.want, i.Render(0), "Render()")
		})
	}
}

func TestChain_RenderMarkdown(t *testing.T) {
	t.Run("Checklist", func(t *testing.T) {
		chain := Chain{
			Source:  TestIssue,
			Current: TestIssue,
			Items: []ChainItem{
				{
					IsCurrent: true,
					Message:   "#12",
					ItemState: Unchecked,
				},
				{
					IsCurrent: false,
					Message:   "#34",
					ItemState: Checked,
				},
				{
					IsCurrent: false,
					Message:   "#56",
					ItemState: Checked,
				},
			},
		}

		expected := `<!-- chainlink generated from https://github.com/RoryQ/gh-chainlink/issues/1 --> 
- [ ] #12 &larr; you are here 
- [x] #34 
- [x] #56`
		assert.Equal(t, expected, chain.RenderMarkdown())
	})

	t.Run("Numbered List", func(t *testing.T) {
		chain := Chain{
			Source:  TestIssue,
			Current: TestIssue,
			Items: []ChainItem{
				{
					IsCurrent: true,
					Message:   "#12",
					ItemState: Numbered,
				},
				{
					IsCurrent: false,
					Message:   "#34",
					ItemState: Numbered,
				},
				{
					IsCurrent: false,
					Message:   "#56",
					ItemState: Numbered,
				},
			},
		}

		expected := `<!-- chainlink generated from https://github.com/RoryQ/gh-chainlink/issues/1 --> 
1. #12 &larr; you are here 
2. #34 
3. #56`
		assert.Equal(t, expected, chain.RenderMarkdown())
	})

	t.Run("Bulleted List", func(t *testing.T) {
		chain := Chain{
			Source:  TestIssue,
			Current: TestIssue,
			Items: []ChainItem{
				{
					IsCurrent: true,
					Message:   "#12",
					ItemState: Bulleted,
				},
				{
					IsCurrent: false,
					Message:   "#34",
					ItemState: Bulleted,
				},
				{
					IsCurrent: false,
					Message:   "#56",
					ItemState: Bulleted,
				},
			},
		}

		expected := `<!-- chainlink generated from https://github.com/RoryQ/gh-chainlink/issues/1 --> 
- #12 &larr; you are here 
- #34 
- #56`
		assert.Equal(t, expected, chain.RenderMarkdown())
	})

	t.Run("List With Header", func(t *testing.T) {
		chain := Chain{
			Header:  "### PR Chain",
			Source:  TestIssue,
			Current: TestIssue,
			Items: []ChainItem{
				{
					IsCurrent: true,
					Message:   "#12",
					ItemState: Bulleted,
				},
				{
					IsCurrent: false,
					Message:   "#34",
					ItemState: Bulleted,
				},
				{
					IsCurrent: false,
					Message:   "#56",
					ItemState: Bulleted,
				},
			},
		}

		expected := `### PR Chain
<!-- chainlink generated from https://github.com/RoryQ/gh-chainlink/issues/1 --> 
- #12 &larr; you are here 
- #34 
- #56`
		assert.Equal(t, expected, chain.RenderMarkdown())
	})
}
