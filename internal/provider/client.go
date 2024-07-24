package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type LineApiClient struct {
	HttpClient     *http.Client
	ChannelId      string
	ChannelSecret  string
	Endpoint       string
	AccessToken    string
	TokenExpiresAt time.Time
}

type StatelessChannelAccessTokenV3Response struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func LineMessagingAPIClient(channel_id string, channel_secret string) (*LineApiClient, error) {
	c := LineApiClient{
		HttpClient:    &http.Client{Timeout: 10 * time.Second},
		Endpoint:      "https://api.line.me/",
		ChannelId:     channel_id,
		ChannelSecret: channel_secret,
	}
	return &c, nil
}

func (c *LineApiClient) GetStatelessChannelAccessTokenV3() (string, error) {

	if time.Now().Before(c.TokenExpiresAt) {
		return c.AccessToken, nil
	}

	oauth_url := c.Endpoint + "oauth2/v3/token"

	data := url.Values{
		"grant_type":    []string{"client_credentials"},
		"client_id":     []string{c.ChannelId},
		"client_secret": []string{c.ChannelSecret},
	}

	print("xxx", "-->"+c.ChannelId+"<--")

	req, err := http.NewRequest("POST", oauth_url, bytes.NewBufferString(data.Encode()))

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HttpClient.Do(req)

	if err != nil {

		return "", err
	}

	defer resp.Body.Close()

	// レスポンスの内容を読み取る
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse StatelessChannelAccessTokenV3Response

	println("body", string(body))

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return "", err
	}

	if tokenResponse.TokenType != "Bearer" {
		return "", fmt.Errorf("unexpected token type: %s", tokenResponse.TokenType)
	}

	c.AccessToken = tokenResponse.AccessToken
	c.TokenExpiresAt = time.Now().Add(time.Second * time.Duration(tokenResponse.ExpiresIn))

	return c.AccessToken, nil
}
