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

	req, err := http.NewRequest("POST", oauth_url, bytes.NewBufferString(data.Encode()))

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HttpClient.Do(req)

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	// レスポンスの内容を読み取る
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse StatelessChannelAccessTokenV3Response

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

type LiffAppsListResponseItemView struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	ModuleMode *bool  `json:"moduleMode,omitempty"`
}

type LiffAppsListResponseItemViewFeatures struct {
	BLE        bool  `json:"ble"`
	QRCode     bool  `json:"qrCode"`
	ModuleMode *bool `json:"moduleMode,omitempty"`
}

type LiffAppsListResponseItem struct {
	LiffId               string                                `json:"liffId"`
	View                 LiffAppsListResponseItemView          `json:"view"`
	Description          *string                               `json:"description,omitempty"`
	PermanentLinkPattern string                                `json:"permanentLinkPattern"`
	Features             *LiffAppsListResponseItemViewFeatures `json:"features,omitempty"`
	Scope                []string                              `json:"scope"`
	BotPrompt            string                                `json:"botPrompt"`
}

type LiffAppsListResponse struct {
	Apps []LiffAppsListResponseItem `json:"apps"`
}

func (c *LineApiClient) ListLiffApps() ([]LiffAppsListResponseItem, error) {
	accessToken, err := c.GetStatelessChannelAccessTokenV3()
	if err != nil {
		return nil, err
	}

	url := c.Endpoint + "liff/v1/apps"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.HttpClient.Do(req)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var liffAppsListResponse LiffAppsListResponse
	err = json.Unmarshal(body, &liffAppsListResponse)
	if err != nil {
		return nil, err
	}
	return liffAppsListResponse.Apps, nil
}

func (c *LineApiClient) GetLiffApp(liffId string) (*LiffAppsListResponseItem, error) {
	liffApps, err := c.ListLiffApps()
	if err != nil {
		return nil, err
	}

	var target LiffAppsListResponseItem
	found := false

	for _, liffApp := range liffApps {
		if liffApp.LiffId == liffId {
			target = liffApp
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("LIFF app not found")
	}
	return &target, nil
}

type LiffAppCreateRequestView struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	ModuleMode *bool  `json:"moduleMode,omitempty"`
}

type LiffAppCreateRequestFeatures struct {
	QRCode *bool `json:"qrCode"`
}

type LiffAppCreateRequest struct {
	View                 LiffAppCreateRequestView      `json:"view"`
	Description          *string                       `json:"description,omitempty"`
	Features             *LiffAppCreateRequestFeatures `json:"features,omitempty"`
	PermanentLinkPattern *string                       `json:"permanentLinkPattern,omitempty"`
	Scope                *[]string                     `json:"scope,omitempty"`
	BotPrompt            *string                       `json:"botPrompt,omitempty"`
}

type LiffAppCreateResponse struct {
	LiffId string `json:"liffId"`
}

func (c *LineApiClient) CreateLiffApp(request LiffAppCreateRequest) (string, error) {

	accessToken, err := c.GetStatelessChannelAccessTokenV3()
	if err != nil {
		return "", err
	}

	url := c.Endpoint + "liff/v1/apps"
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	println("Request body: %s", string(reqBody))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var createLiffAppResponse LiffAppCreateResponse
	err = json.Unmarshal(body, &createLiffAppResponse)
	if err != nil {
		return "", err
	}
	return createLiffAppResponse.LiffId, nil
}

type LiffAppUpdateRequestView struct {
	Type       *string `json:"type"`
	URL        *string `json:"url"`
	ModuleMode *bool   `json:"moduleMode,omitempty"`
}

type LiffAppUpdateRequestFeatures struct {
	QRCode *bool `json:"qrCode"`
}

type LiffAppUpdateRequest struct {
	View                 LiffAppUpdateRequestView      `json:"view"`
	Description          *string                       `json:"description,omitempty"`
	Features             *LiffAppUpdateRequestFeatures `json:"features,omitempty"`
	PermanentLinkPattern *string                       `json:"permanentLinkPattern,omitempty"`
	Scope                *[]string                     `json:"scope,omitempty"`
	BotPrompt            *string                       `json:"botPrompt,omitempty"`
}

func (c *LineApiClient) UpdateLiffApp(liffId string, request LiffAppUpdateRequest) error {

	accessToken, err := c.GetStatelessChannelAccessTokenV3()
	if err != nil {
		return err
	}

	url := c.Endpoint + "liff/v1/apps/" + liffId
	reqBody, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
