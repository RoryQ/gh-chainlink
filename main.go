package main

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/kr/pretty"
)

func main() {
	fmt.Println("hi world, this is the gh-chainlink extension!")
	client, err := api.DefaultRESTClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	{
		response := struct{ Login string }{}
		err = client.Get("user", &response)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("running as %s\n", response.Login)
	}

	{
		repo := must(repository.Current())
		current := ChainIssue{
			Repo:   repo,
			Number: 1,
		}
		response := struct{ Body string }{}
		must0(client.Get(current.Path(), &response))
		fmt.Println(response.Body)

		pretty.Print(Parse(current, response.Body))
	}
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
