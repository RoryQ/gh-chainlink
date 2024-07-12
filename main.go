package main

import "log/slog"

func main() {
	// Detect repo and issue for current branch
	client := must(NewGhClient())

	// Use provided issue ref if provided
	currentIssue := ChainIssue{
		Repo:   client.currentRepo,
		Number: 1,
	}

	// get chain from ref issue
	issue := must(client.GetIssue(currentIssue.Number))
	chain := must(Parse(currentIssue, issue.Body))

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
