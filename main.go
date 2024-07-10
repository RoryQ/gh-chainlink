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

	println(chain.RenderMarkdown())

	// Upsert each linked issue with current chain, skipping current item
	for _, item := range chain.Items {
		if item.HostPath() == currentIssue.HostPath() {
			continue
		}

		slog.Info("upsert chain for issue", "issue", item.Number, "hostPath", item.HostPath())

		// itemIssue := must(client.GetIssue(item.Number))
		// must0(client.UpdateIssueBody(item.Number, itemIssue.Body))
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
