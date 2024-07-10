package main

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/cli/go-gh/v2/pkg/repository"
)

type Chain struct {
	Header  string
	Current ChainIssue
	Items   []ChainItem
	Raw     string
}

type ChainItem struct {
	ChainIssue
	IsCurrent bool
	Message   string
	Checked   bool
	Raw       string
}

type ChainIssue struct {
	Repo   repository.Repository
	Number int
}

func (i ChainIssue) Path() string {
	return fmt.Sprintf("repos/%s/%s/issues/%d", i.Repo.Owner, i.Repo.Name, i.Number)
}

func (i ChainIssue) HostPath() string {
	return fmt.Sprintf("%s/repos/%s/%s/issues/%d", i.Repo.Host, i.Repo.Owner, i.Repo.Name, i.Number)
}

func (i ChainItem) Render() string {
	return fmt.Sprintf("- [%s] %s %s",
		iif(i.Checked, "x", " "),
		i.Message,
		iif(i.IsCurrent, ":arrow_left: This PR", ""),
	)
}

func iif[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func (c Chain) ResetCurrent(to ChainIssue) Chain {
	newChain := c
	newChain.Current = to
	for i := range c.Items {
		c.Items[i].IsCurrent = false
		if c.Items[i].ChainIssue.HostPath() == to.HostPath() {
			c.Items[i].IsCurrent = true
		}
	}
	return newChain
}

func (c Chain) RenderMarkdown() string {
	templateString := `<!--chainlink-->
{{- range .Items }} 
{{.Render}} {{- end}}
`

	tmpl := template.Must(template.New("").Parse(templateString))
	buf := new(bytes.Buffer)
	must0(tmpl.Execute(buf, c))
	return buf.String()
}
