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
	Source  ChainIssue
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

const (
	CurrentIndicator = "&larr; you are here"
)

type ItemState int

const (
	Unchecked ItemState = iota
	Checked
	Numbered
	Bulleted
)

type ChainIssue struct {
	Repo          repository.Repository
	Number        int
	IsPullRequest bool
}

func (i ChainIssue) Path() string {
	if i.IsPullRequest {
		return fmt.Sprint("repos/", i.Repo.Owner, "/", i.Repo.Name, "/pulls/", i.Number)
	}
	return fmt.Sprint("repos/", i.Repo.Owner, "/", i.Repo.Name, "/issues/", i.Number)
}

func (i ChainIssue) HostPath() string {
	return fmt.Sprint(i.Repo.Host, "/", i.Path())
}

func (i ChainIssue) IsSame(other ChainIssue) bool {
	return i.Repo == other.Repo && i.Number == other.Number
}

func (i ChainItem) Render(pointIndex int) string {
	rendered := fmt.Sprintln(
		i.renderListPoint(pointIndex),
		i.Message,
		iif(i.IsCurrent, CurrentIndicator, ""))
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
		if c.Items[i].ChainIssue.IsSame(to) {
			c.Items[i].IsCurrent = true
		}
	}
	return newChain
}

func (c Chain) RenderMarkdown() string {
	templateString := `{{- if .Header }}{{ println .Header }}{{ end -}}
<!-- chainlink generated from {{.Source.HostPath}} -->
{{- range $i, $v :=  .Items }} 
{{$v.Render $i }} {{- end}}`

	tmpl := template.Must(template.New("").Parse(templateString))
	buf := new(bytes.Buffer)
	must0(tmpl.Execute(buf, c))
	return buf.String()
}
