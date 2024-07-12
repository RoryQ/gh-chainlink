package main

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

		println(chain.ResetCurrent(item.ChainIssue).RenderMarkdown())

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
