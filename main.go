package main

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func main() {
	fmt.Println("hi world, this is the gh-chainlink extension!")
	client := must(api.DefaultRESTClient())

	repo := must(repository.Current())
	current := ChainIssue{
		Repo:   repo,
		Number: 1,
	}
	response := repoResponse{}
	must0(client.Get(current.Path(), &response))
	fmt.Println(response.Body)

	chain := must(Parse(current, response.Body))

	// followedLinks := map[string]Chain{}
	for _, item := range chain.Items {
		if item.IsCurrent {
			continue
		}

		resp := repoResponse{}
		must0(client.Get(item.Path(), &resp))

		fmt.Printf("%+v\n", resp)
	}
}

type repoResponse struct {
	Title  string
	Body   string
	Number int
	State  string
}

// For more examples of using go-gh, see:
// https://github.com/cli/go-gh/blob/trunk/example_gh_test.go

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
