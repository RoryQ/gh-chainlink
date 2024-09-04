package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type GhClient struct {
	clientLookup  map[string]*api.RESTClient
	currentRepo   repository.Repository
	currentBranch string
}

func (c *GhClient) InGitRepo() bool {
	return c.currentRepo != (repository.Repository{})
}

func NewGhClient() (*GhClient, error) {
	host, _ := auth.DefaultHost()
	apiClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}

	current, _ := repository.Current()
	apiClientHostLookup := map[string]*api.RESTClient{
		host: apiClient,
	}

	return &GhClient{clientLookup: apiClientHostLookup, currentRepo: current}, nil
}

type IssueResponse struct {
	Title  string
	Body   string
	Number int
	State  string
	Url    string
}

func (c *GhClient) GetIssue(issue ChainIssue) (IssueResponse, error) {
	response := IssueResponse{}
	client, err := c.getClient(issue.Repo.Host)
	if err != nil {
		return IssueResponse{}, err
	}
	err = client.Get(issue.Path(), &response)
	if err != nil {
		return IssueResponse{}, err
	}
	return response, nil
}

func (c *GhClient) IsPull(issue ChainIssue) bool {
	apiPath := fmt.Sprintf("repos/%s/%s/issues/%d", issue.Repo.Owner, issue.Repo.Name, issue.Number)
	response := IssueResponse{}
	client, err := c.getClient(issue.Repo.Host)
	if err != nil {
		slog.Error("Error getting client", "error", err)
		return false
	}
	err = client.Get(apiPath, &response)
	he := &api.HTTPError{}
	if errors.As(err, &he) {
		return he.StatusCode != http.StatusNotFound
	}
	if err != nil {
		slog.Error("Error getting issue", "error", err)
		return false
	}
	return true
}

func (c *GhClient) UpdateIssueBody(issue ChainIssue, body string) error {
	response := map[string]any{}
	request, err := c.encodeJson(map[string]any{"body": body})
	if err != nil {
		return err
	}
	err = c.clientLookup[issue.Repo.Host].Patch(issue.Path(), request, &response)
	return err
}

func (c *GhClient) encodeJson(request map[string]any) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(request); err != nil {
		return nil, err
	}

	return buf, nil
}

func (c *GhClient) getClient(host string) (*api.RESTClient, error) {
	if client, ok := c.clientLookup[host]; ok {
		return client, nil
	}

	client, err := api.NewRESTClient(api.ClientOptions{
		Host:    host,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	c.clientLookup[host] = client
	return client, nil
}
