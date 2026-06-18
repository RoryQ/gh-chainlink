package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/fatih/color"
	"github.com/sourcegraph/conc/pool"
)

var (
	bold    = color.New(color.Bold).SprintFunc()
	hiBlack = color.New(color.FgHiBlack).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	blue    = color.New(color.FgHiBlue).SprintFunc()
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(color.Output, "%s\n\n", "Chainlink - link chained pull requests and issues.")
		fmt.Fprintf(color.Output, "%s\n", bold("USAGE"))
		fmt.Fprintf(color.Output, "  %s\n", "gh chainlink <issue ref>")
		fmt.Fprintf(color.Output, "  %s\n\n", "gh chainlink append <first issue ref> <issue ref> ...")
		fmt.Fprintf(color.Output, "%s\n", bold("COMMANDS"))
		fmt.Fprintf(color.Output, "  %s\n\n", "append: Lookup the chain in the first issue/PR, append the rest, and sync them.")
		fmt.Fprintf(color.Output, "%s", bold("ISSUE REF"))
		fmt.Fprintf(color.Output, "%s\n", `
  autodetect: Leave empty to use the pull request for the current branch.
  number:   Enter the issue or pull request number for the current repo e.g. 123.
  url: Enter the issue or pull request url e.g. https://github.com/RoryQ/gh-chainlink/issues/1
  `)
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()

	// Detect repo and issue for current branch
	client := must(NewGhClient())

	if len(args) > 0 && args[0] == "append" {
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: 'append' requires at least two issue/PR references (the target first issue/PR, and the issue/PRs to append).\n\n")
			flag.Usage()
			os.Exit(1)
		}

		firstIssue := parseIssueArg(args[1], client.currentRepo)
		if firstIssue.Number == 0 {
			fmt.Fprintf(os.Stderr, "Error: Invalid first issue reference %q\n", args[1])
			os.Exit(1)
		}

		issue, err := client.GetIssue(firstIssue)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching first issue: %v\n", err)
			os.Exit(1)
		}

		chain, err := Parse(firstIssue, issue.Body)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				chain = &Chain{
					Header:  "## PR Chain",
					Source:  firstIssue,
					Current: firstIssue,
					Items: []ChainItem{
						{
							ChainIssue: firstIssue,
							IsCurrent:  true,
							Message:    fmt.Sprintf("#%d", firstIssue.Number),
							ItemState:  Numbered,
						},
					},
				}
			} else {
				fmt.Fprintf(os.Stderr, "Error parsing issue body: %v\n", err)
				os.Exit(1)
			}
		}

		for _, arg := range args[2:] {
			appendedIssue := parseIssueArg(arg, client.currentRepo)
			if appendedIssue.Number == 0 {
				fmt.Fprintf(os.Stderr, "Error: Invalid issue reference to append %q\n", arg)
				os.Exit(1)
			}

			if !chain.Contains(appendedIssue) {
				message := fmt.Sprintf("#%d", appendedIssue.Number)
				if appendedIssue.Repo != chain.Source.Repo {
					message = appendedIssue.URL()
				}

				itemState := Numbered
				if len(chain.Items) > 0 {
					itemState = chain.Items[len(chain.Items)-1].ItemState
				}

				chain.Items = append(chain.Items, ChainItem{
					ChainIssue: appendedIssue,
					IsCurrent:  appendedIssue.IsSame(chain.Current),
					Message:    message,
					ItemState:  itemState,
				})
			}
		}

		_, err = tea.NewProgram(model{
			gh:        client,
			sub:       make(chan responseMsg),
			responses: make(map[int]responseMsg),
			chain:     *chain,
		}).Run()

		if err != nil {
			slog.Error("Error running program", "error", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	// Use provided issue ref if provided
	targetIssue := getTargetIssue(args)

	if targetIssue.Number == 0 {
		flag.Usage()
		os.Exit(0)
	}

	// get chain from ref issue
	issue := must(client.GetIssue(targetIssue))
	chain := must(Parse(targetIssue, issue.Body))

	_, err := tea.NewProgram(model{
		gh:        client,
		sub:       make(chan responseMsg),
		responses: make(map[int]responseMsg),
		chain:     *chain,
	}).Run()

	if err != nil {
		slog.Error("Error running program", "error", err)
		os.Exit(1)
	}
}

func updateIssue(client *GhClient, chain Chain, item ChainItem) (string, error) {
	item.IsPullRequest = client.IsPull(item.ChainIssue)
	// update the CurrentLocationIndicator to the current issue
	issueChainString := chain.ResetCurrent(item.ChainIssue).RenderMarkdown()

	itemIssue, err := client.GetIssue(item.ChainIssue)
	if err != nil {
		return "error", fmt.Errorf("error retrieving item %d: %w", item.Number, err)
	}

	updatedBody := ReplaceChain(itemIssue.Body, issueChainString)
	if updatedBody == itemIssue.Body {
		return "skipped", nil
	}

	if err := client.UpdateIssueBody(item.ChainIssue, updatedBody); err != nil {
		return "error", fmt.Errorf("error updating item %d: %w", item.Number, err)
	}

	return "updated", nil
}

func parseIssueArg(arg string, currentRepo repository.Repository) ChainIssue {
	issueRef := arg
	// current repo reference if number only
	if _, err := strconv.Atoi(issueRef); err == nil {
		issueRef = "#" + issueRef
	}

	issue := issueFromString(issueRef)
	// Argument was a URL
	if issue.Number != 0 && issue.Repo.Host != "" {
		return issue
	}

	// Argument was a number
	issue.Repo = currentRepo
	return issue
}

func getTargetIssue(args []string) ChainIssue {
	currentRepo, _ := repository.Current()
	// use first argument
	if len(args) >= 1 {
		return parseIssueArg(args[0], currentRepo)
	}


	// detect from branch
	stdOut, stdErr, err := gh.Exec("pr", "status", "--json", "number,baseRefName,url")
	if err != nil {
		panic(err)
	}
	println(stdErr.String())

	jsonResp := struct {
		CurrentBranch struct {
			BaseRefName string `json:"baseRefName"`
			Number      int    `json:"number"`
			Url         string `json:"url"`
		}
	}{}
	must0(json.Unmarshal(stdOut.Bytes(), &jsonResp))

	return issueFromMessage(currentRepo, jsonResp.CurrentBranch.Url)
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func must0(err error) {
	if err != nil {
		panic(err)
	}
}

type responseMsg struct {
	index  int
	result string
	err    error
}

func (m model) updatePRs() tea.Cmd {
	return func() tea.Msg {
		p := pool.New().WithMaxGoroutines(5)
		for i, item := range m.chain.Items {
			i, item := i, item
			p.Go(func() {
				resp, err := updateIssue(m.gh, m.chain, item)
				m.sub <- responseMsg{index: i, result: resp, err: err}
			})
		}
		p.Wait()
		return nil
	}
}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan responseMsg) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

type model struct {
	gh        *GhClient
	sub       chan responseMsg
	responses map[int]responseMsg
	chain     Chain
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.updatePRs(),          // generate activity
		waitForActivity(m.sub), // wait for activity
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case responseMsg:
		m.responses[v.index] = v
		if len(m.responses) == len(m.chain.Items) {
			return m, tea.Quit
		}
		return m, waitForActivity(m.sub) // wait for next event
	default:
		return m, nil
	}
}

func (m model) View() string {
	sb := new(strings.Builder)
	if m.chain.Header != "" {
		_, _ = fmt.Fprintln(sb, blue(m.chain.Header))
	}
	for i, item := range m.chain.Items {
		if response, ok := m.responses[i]; ok {
			switch response.result {
			case "updated":
				_, _ = fmt.Fprintln(sb, green("✓"), item.renderListPoint(i), item.Message)
			case "skipped":
				_, _ = fmt.Fprintln(sb, yellow("∅"), item.renderListPoint(i), item.Message)
			case "error":
				_, _ = fmt.Fprintln(sb, red("✗"), item.renderListPoint(i), item.Message, red(response.err))
			}
		} else {
			_, _ = fmt.Fprintln(sb, hiBlack("_"), item.renderListPoint(i), item.Message)
		}
	}
	return sb.String()
}
