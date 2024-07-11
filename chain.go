package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
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
	ItemState ItemState
	Raw       string
}

const CurrentPrIndicator = "&larr; This PR"

type ItemState int

const (
	Unchecked ItemState = iota
	Checked
	Numbered
	Bulleted
)

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

func (i ChainItem) Render(pointIndex int) string {
	rendered := fmt.Sprintln(
		i.renderListPoint(pointIndex),
		i.Message,
		iif(i.IsCurrent, CurrentPrIndicator, ""))
	return strings.TrimRight(rendered, " \n")
}

func (i ChainItem) renderListPoint(pointIndex int) string {
	switch i.ItemState {
	case Numbered:
		return strconv.Itoa(pointIndex+1) + "."
	case Checked:
		return "- [x]"
	case Unchecked:
		return "- [ ]"
	case Bulleted:
		return "-"
	}
	panic("unreachable")
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
{{- range $i, $v :=  .Items }} 
{{$v.Render $i }} {{- end}}
`

	tmpl := template.Must(template.New("").Parse(templateString))
	buf := new(bytes.Buffer)
	must0(tmpl.Execute(buf, c))
	return buf.String()
}
