package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/fatih/color"
)

var (
	bold    = color.New(color.Bold).SprintFunc()
	hiBlack = color.New(color.FgHiBlack).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(color.Output, "%s\n\n", "Chainlink - link chained pull requests and issues.")
		fmt.Fprintf(color.Output, "%s\n", bold("USAGE"))
		fmt.Fprintf(color.Output, "  %s\n\n", "gh chainlink <issue ref>")
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

	// Use provided issue ref if provided
	targetIssue := getTargetIssue(args)

	if targetIssue.Number == 0 {
		flag.Usage()
		os.Exit(0)
	}

	// get chain from ref issue
	issue := must(client.GetIssue(targetIssue.Number))
	chain := must(Parse(targetIssue, issue.Body))

	// Upsert each linked issue with current chain, skipping current item
	for i, item := range chain.Items {
		chain.Items[i].IsPullRequest = client.IsPull(item.Number)

		issueChainString := chain.ResetCurrent(item.ChainIssue).RenderMarkdown()

		itemIssue, err := client.GetIssue(item.Number)
		if err != nil {
			slog.Error("Error retrieving item", "number", item.Number, "error", err)
			continue
		}

		updatedBody := ReplaceChain(itemIssue.Body, issueChainString)

		if updatedBody != itemIssue.Body {
			err := client.UpdateIssueBody(item.Number, updatedBody)
			if err != nil {
				slog.Error("Error updating item", "number", item.Number, "error", err)
			}
		}
	}
}

func getTargetIssue(args []string) ChainIssue {
	currentRepo := must(repository.Current())
	// use first argument
	if len(args) >= 1 {
		issueRef := args[0]

		// current repo reference if number only
		if _, err := strconv.Atoi(issueRef); err == nil {
			issueRef = "#" + issueRef
		}

		return issueFromMessage(currentRepo, issueRef)
	}

	// detect from branch
	stdOut, stdErr, err := gh.Exec("pr", "status", "--json", "number,baseRefName")
	if err != nil {
		panic(err)
	}

	println(stdErr.String())

	jsonResp := struct {
		CurrentBranch struct {
			BaseRefName string
			Number      string
			Url         string
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
