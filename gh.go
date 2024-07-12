package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type GhClient struct {
	RESTClient    *api.RESTClient
	currentRepo   repository.Repository
	currentBranch string
}

func NewGhClient() (*GhClient, error) {
	apiClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	current, err := repository.Current()
	if err != nil {
		return nil, err
	}
	return &GhClient{RESTClient: apiClient, currentRepo: current}, nil
}

type IssueResponse struct {
	Title  string
	Body   string
	Number int
	State  string
	Url    string
}

func (c *GhClient) GetIssue(num int) (IssueResponse, error) {
	apiPath := fmt.Sprintf("repos/%s/%s/issues/%d", c.currentRepo.Owner, c.currentRepo.Name, num)
	response := IssueResponse{}
	err := c.RESTClient.Get(apiPath, &response)
	if err != nil {
		return IssueResponse{}, err
	}
	return response, nil
}

func (c *GhClient) IsPull(num int) bool {
	apiPath := fmt.Sprintf("repos/%s/%s/pulls/%d", c.currentRepo.Owner, c.currentRepo.Name, num)
	response := IssueResponse{}
	err := c.RESTClient.Get(apiPath, &response)
	he := &api.HTTPError{}
	if errors.As(err, &he) {
		return he.StatusCode != http.StatusNotFound
	}
	return true
}

func (c *GhClient) UpdateIssueBody(num int, body string) error {
	apiPath := fmt.Sprintf("repos/%s/%s/issues/%d", c.currentRepo.Owner, c.currentRepo.Name, num)
	response := map[string]any{}
	request, err := c.encodeJson(map[string]any{"body": body})
	if err != nil {
		return err
	}
	err = c.RESTClient.Patch(apiPath, request, &response)
	slog.Info("patch response", "response", response, "error", err)
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
