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
			Header:  closestValidHeaderTo(content, indLineNumber).Raw,
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

		start := indLineNumber
		// start from header if it was found in body and replacement chain
		if closestValidHeaderTo(chain, findRE(chain, indicatorRE)[0].LineNumber).Raw != "" {
			header := closestValidHeaderTo(body, indLineNumber)
			if header.Raw != "" {
				start = header.LineNumber
			}
		}

		// end at bottom of checklist
		checklistForIndicator := checklists[c]
		checklistEndLineNumber := checklistForIndicator.LineNumbers[len(checklistForIndicator.LineNumbers)-1]

		// remove and insert new checklist
		body = removeLines(body, start, checklistEndLineNumber)
		return insertLinesAt(body, start, chain)
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
	lines = slicesConcat(lines[:at], withLines, lines[at+1:])
	return strings.Join(lines, "\n")
}

func blockToItems(current ChainIssue, b block) (items []ChainItem) {
	for _, line := range strings.Split(b.Raw, "\n") {
		matches, ok := FindMatchGroups(itemRE, line)
		if !ok {
			continue
		}
		message := parseMessage(matches)
		issue := issueFromMessage(current.Repo, message)
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

func issueFromMessage(currentRepo repository.Repository, s string) ChainIssue {
	issue := issueFromString(s)
	if issue.Repo == (repository.Repository{}) {
		issue.Repo = currentRepo
	}
	return issue
}

func issueFromString(s string) ChainIssue {
	urlRE := regexp.MustCompile(`(?:https?://(?P<host>[^/]+)/(?P<owner>[^/]+)/(?P<repo>[^/]+)/(issues|pull)/(?P<number>\d+).*)`)
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

func closestValidHeaderTo(content string, indLineNumber int) reMatch {
	headers := findRE(content, headerRE)
	if len(headers) == 0 {
		return reMatch{LineNumber: -1}
	}

	h := sort.Search(len(headers), func(i int) bool {
		return headers[i].LineNumber > indLineNumber
	})
	closest := headers[h-1]

	lines := strings.Split(content, "\n")
	for i := closest.LineNumber + 1; i < indLineNumber; i++ {
		// check all lines between header and indicator are empty
		if len(strings.TrimSpace(lines[i])) > 0 {
			return reMatch{LineNumber: -1}
		}
	}
	return closest
}

// Concat returns a new slice concatenating the passed in slices.
func slicesConcat[S ~[]E, E any](ss ...S) S {
	size := 0
	for _, s := range ss {
		size += len(s)
		if size < 0 {
			panic("len out of range")
		}
	}
	newslice := slices.Grow[S](nil, size)
	for _, s := range ss {
		newslice = append(newslice, s...)
	}
	return newslice
}
