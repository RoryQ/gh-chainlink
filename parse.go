package main

import (
	"errors"
	"log/slog"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

type reMatch struct {
	LineNumber int
	Raw        string
}

var (
	indicatorRE = regexp.MustCompile(`(?i)<!--\s*chainlink(?:\s*| generated from.*)-->`)
	headerRE    = regexp.MustCompile(`(?im)^ {0,3}#{1,6}\s.*`)
	itemRE      = regexp.MustCompile(`(?i)^\s{0,4}(- (?P<Checked>\[[ x]])?|(?P<Numbered>\d+)[.] )(:? *)(?P<Message>.*)`)
	ErrNotFound = errors.New("no chainlink list found")
)

func Parse(current ChainIssue, content string) (*Chain, error) {
	indicators := findRE(content, indicatorRE)
	if len(indicators) == 0 {
		return nil, ErrNotFound
	}

	checklists := findChecklistBlocks(content)
	headers := findRE(content, headerRE)
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
			Header:  closestHeaderTo(headers, indLineNumber),
			Source:  current,
			Current: current,
			Items:   blockToItems(current, checklistForIndicator),
			Raw:     checklistForIndicator.Raw,
		}, nil
	}

	return nil, ErrNotFound
}

func FindMatchGroups(re *regexp.Regexp, s string) (map[string]string, bool) {
	getNamedMatches := func(re *regexp.Regexp, matches []string) map[string]string {
		result := make(map[string]string)
		for i, name := range re.SubexpNames() {
			if i < len(matches) {
				result[name] = matches[i]
			}
		}
		return result
	}
	matches := re.FindStringSubmatch(s)
	return getNamedMatches(re, matches), len(matches) > 0
}

func ReplaceChain(body, chain string) string {
	indicators := findRE(body, indicatorRE)
	if len(indicators) == 0 {
		// not found so append the chain to the current body
		return body + "\n" + chain
	}

	checklists := findChecklistBlocks(body)
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
		body = removeLines(body, indLineNumber, checklistForIndicator.LineNumbers[len(checklistForIndicator.LineNumbers)-1])
		return insertLinesAt(body, indLineNumber, chain)
	}

	return strings.ReplaceAll(body, indicators[0].Raw, chain)
}

func removeLines(s string, start, end int) string {
	lines := strings.Split(s, "\n")
	lines = append(lines[:start], lines[end:]...)
	return strings.Join(lines, "\n")
}

func insertLinesAt(s string, at int, with string) string {
	lines := strings.Split(s, "\n")
	withLines := strings.Split(with, "\n")
	lines = slices.Concat(lines[:at], withLines, lines[at+1:])
	return strings.Join(lines, "\n")
}

func blockToItems(current ChainIssue, b block) (items []ChainItem) {
	for _, line := range strings.Split(b.Raw, "\n") {
		matches, ok := FindMatchGroups(itemRE, line)
		if !ok {
			continue
		}
		message := parseMessage(matches)
		issue := issueFromMessage(current, message)
		items = append(items,
			ChainItem{
				ChainIssue: issue,
				IsCurrent:  issue == current,
				Message:    message,
				ItemState:  parseItemState(matches),
				Raw:        line,
			},
		)
	}

	return
}

func parseItemState(s map[string]string) ItemState {
	if checked, ok := s["Checked"]; ok && checked != "" {
		isChecked := strings.EqualFold(checked, "[x]")
		if isChecked {
			return Checked
		}
		return Unchecked
	}

	if numbered, ok := s["Numbered"]; ok && numbered != "" {
		return Numbered
	}

	return Bulleted
}

func parseMessage(s map[string]string) string {
	trimIndicator := func(str string) string {
		return strings.TrimSuffix(str, CurrentIndicator)
	}
	return strings.TrimSpace(trimIndicator(strings.TrimSpace(s["Message"])))
}

type stringMutFunc = func(string) string

func issueFromMessage(current ChainIssue, s string) ChainIssue {
	urlRE := regexp.MustCompile(`(?:https?://(?P<host>[^/]+)/(?P<owner>[^/]+)/(?P<repo>[^/]+)/issues/(?P<number>\d+).*)`)
	numberRE := regexp.MustCompile(`(?:#(?P<number>\d+))`)

	atoi := func(s string) int {
		i, _ := strconv.Atoi(s)
		return i
	}

	s = strings.TrimSpace(s)

	if urlMatch, matched := FindMatchGroups(urlRE, s); matched {
		return ChainIssue{
			Repo: repository.Repository{
				Host:  urlMatch["host"],
				Owner: urlMatch["owner"],
				Name:  urlMatch["repo"],
			},
			Number: atoi(urlMatch["number"]),
		}
	}

	if numberMatch, matched := FindMatchGroups(numberRE, s); matched {
		return ChainIssue{
			Repo:   current.Repo,
			Number: atoi(numberMatch["number"]),
		}
	}

	return ChainIssue{}
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
	matches := findRE(content, itemRE)

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

func closestHeaderTo(headers []reMatch, indLineNumber int) string {
	if len(headers) == 0 {
		return ""
	}

	h := sort.Search(len(headers), func(i int) bool {
		return headers[i].LineNumber > indLineNumber
	})

	foundHeader := headers[h-1].Raw
	return foundHeader
}
