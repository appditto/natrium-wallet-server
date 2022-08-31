package gql

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

type GQLError string

const (
	InvalidUsernamePasssword GQLError = "Invalid username or password"
	ServerError                       = "Unknown server error, try again later"
)

type BpowClient struct {
	client graphql.Client
	ctx    context.Context
}

type authedTransport struct {
	wrapped http.RoundTripper
	token   string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.token)
	return t.wrapped.RoundTrip(req)
}

func NewBpowClient(url string, token string) *BpowClient {
	return &BpowClient{
		client: graphql.NewClient(url, &http.Client{Transport: &authedTransport{wrapped: http.DefaultTransport, token: token}}),
		ctx:    context.Background(),
	}
}

func (c *BpowClient) WorkGenerate(hash string, difficultyMultipler int) (string, error) {
	resp, err := workGenerate(c.ctx, c.client, WorkGenerateInput{
		Hash:                 hash,
		DifficultyMultiplier: difficultyMultipler,
		BlockAward:           false,
	})

	if err != nil {
		fmt.Printf("Error generating work %v", err)
		return "", err
	}

	return resp.WorkGenerate, nil
}
