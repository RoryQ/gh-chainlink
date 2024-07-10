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
		Checked    bool
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
				Checked:    true,
			},
			want: "- [x] #1234",
		},
		"Current": {
			fields: fields{
				ChainIssue: TestIssue,
				Message:    "#123",
				IsCurrent:  true,
			},
			want: "- [ ] #123 :arrow_left: This PR",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i := ChainItem{
				ChainIssue: tt.fields.ChainIssue,
				IsCurrent:  tt.fields.IsCurrent,
				Message:    tt.fields.Message,
				Checked:    tt.fields.Checked,
				Raw:        tt.fields.Raw,
			}
			assert.Equalf(t, tt.want, i.Render(), "Render()")
		})
	}
}

func TestChain_RenderMarkdown(t *testing.T) {
	t.Run("Checklist", func(t *testing.T) {
		chain := Chain{
			Current: TestIssue,
			Items: []ChainItem{
				{
					IsCurrent: true,
					Message:   "#12",
					Checked:   false,
				},
				{
					IsCurrent: false,
					Message:   "#34",
					Checked:   true,
				},
				{
					IsCurrent: false,
					Message:   "#56",
					Checked:   true,
				},
			},
		}

		expected := `<!--chainlink--> 
- [ ] #12 :arrow_left: This PR 
- [x] #34  
- [x] #56 
`
		assert.Equal(t, expected, chain.RenderMarkdown())
	})
}
