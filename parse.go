package main

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

type Chain struct {
	Current ChainIssue
	Items   []ChainItem
	Raw     string
}

type ChainItem struct {
	ChainIssue
	Message string
	Checked bool
	Raw     string
}

type ChainIssue struct {
	Repo   repository.Repository
	Number int
	Title  string
}

func (i ChainIssue) Path() string {
	return fmt.Sprintf("repos/%s/%s/issues/%d", i.Repo.Owner, i.Repo.Name, i.Number)
}

type reMatch struct {
	LineNumber int
	Raw        string
}

var (
	indicatorRE = regexp.MustCompile(`(?i)<!--\s*Chain(-)?link\s*-->`)
)

func Parse(current ChainIssue, content string) (*Chain, error) {
	indicators := findRE(content, indicatorRE)
	if len(indicators) == 0 {
		return nil, errors.New("no chainlink list found")
	}

	checklists := findChecklistBlocks(content)

	for _, ind := range indicators {
		indLineNumber := ind.LineNumber

		c := sort.Search(len(checklists), func(i int) bool {
			return checklists[i].LineNumbers[0] > indLineNumber
		})

		if c >= len(checklists) {
			slog.Warn("no list found for indicator", "lineNumber", indLineNumber)
			continue
		}

		checklistForIndicator := checklists[c]
		return &Chain{
			Current: current,
			Items:   blockToItems(current, checklistForIndicator),
			Raw:     checklistForIndicator.Raw,
		}, nil
	}

	return nil, errors.New("no chainlink list found")
}

func blockToItems(current ChainIssue, b block) (items []ChainItem) {
	re := regexp.MustCompile(`(?i)- (?P<Checked>\[[ x]]) (?P<Message>.*)`)
	parseChecked := func(s string) bool {
		return strings.EqualFold(s, "[x]")
	}

	for _, line := range strings.Split(b.Raw, "\n") {
		matches := re.FindAllStringSubmatch(line, -1)[0]
		items = append(items,
			ChainItem{
				Message: matches[re.SubexpIndex("Message")],
				Checked: parseChecked(matches[re.SubexpIndex("Checked")]),
				Raw:     line,
			},
		)
	}

	return
}

func findRE(content string, re *regexp.Regexp) []reMatch {
	var matches []reMatch

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, reMatch{LineNumber: i, Raw: line})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool { return j > i })
	return matches
}

type block struct {
	Raw         string
	LineNumbers []int
}

func findChecklistBlocks(content string) (blocks []block) {
	re := regexp.MustCompile(`(?i)- \[[ x]\] .*`)

	matches := findRE(content, re)

	b := block{}
	for _, m := range matches {
		// block start
		if b.Raw == "" {
			b.Raw += m.Raw
			b.LineNumbers = append(b.LineNumbers, m.LineNumber)
			continue
		}

		// block continuing
		last := b.LineNumbers[len(b.LineNumbers)-1]
		if last+1 == m.LineNumber {
			b.Raw += "\n" + m.Raw
			b.LineNumbers = append(b.LineNumbers, m.LineNumber)
			continue
		}

		// block ended
		blocks = append(blocks, b)
		b = block{
			Raw:         m.Raw,
			LineNumbers: []int{m.LineNumber},
		}
	}
	if b.Raw != "" {
		blocks = append(blocks, b)
	}

	return blocks
}
