/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package slack

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Client allows you to provide connection to Slack API Server
// It contains a token that allows to authenticate connection to post and work with channels in the domain
type Client struct {
	token string
}

const (
	apiUrl = "https://slack.com/api/"

	authTest = apiUrl + "auth.test"
	apiTest  = apiUrl + "api.test"

	channelsList = apiUrl + "channels.list"

	chatPostMessage = apiUrl + "chat.postMessage"
)

type APIResponse struct {
	Ok bool `json:"ok"`
}

type AuthResponse struct {
	Ok     bool   `json:"ok"`
	URL    string `json:"url"`
	Team   string `json:"team"`
	User   string `json:"user"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
}

type Channel struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	IsChannel      bool     `json:"is_channel"`
	Created        int      `json:"created"`
	Creator        string   `json:"creator"`
	IsArchived     bool     `json:"is_archived"`
	IsGeneral      bool     `json:"is_general"`
	NameNormalized string   `json:"name_normalized"`
	IsShared       bool     `json:"is_shared"`
	IsOrgShared    bool     `json:"is_org_shared"`
	IsMember       bool     `json:"is_member"`
	Members        []string `json:"members"`
	Topic          struct {
		Value   string `json:"value"`
		Creator string `json:"creator"`
		LastSet int    `json:"last_set"`
	} `json:"topic"`
	Purpose struct {
		Value   string `json:"value"`
		Creator string `json:"creator"`
		LastSet int    `json:"last_set"`
	} `json:"purpose"`
	PreviousNames []interface{} `json:"previous_names"`
	NumMembers    int           `json:"num_members"`
}

type ChannelList struct {
	Ok       bool      `json:"ok"`
	Channels []Channel `json:"channels"`
}

// Create a slack client with an API token.
func NewClient(token string) *Client {
	return &Client{
		token: token,
	}
}

func (sl *Client) VerifyAPI() (bool, error) {
	t, e := sl.postMessage(apiTest, sl.urlValues())
	if e != nil {
		return false, e
	}

	var apiResponse APIResponse
	e = json.Unmarshal(t, &apiResponse)
	if e != nil {
		return false, e
	}
	return apiResponse.Ok, nil
}

func (sl *Client) VerifyAuth() (bool, error) {
	t, e := sl.postMessage(authTest, sl.urlValues())
	if e != nil {
		return false, e
	}

	var authResponse AuthResponse
	e = json.Unmarshal(t, &authResponse)
	if e != nil {
		return false, e
	}
	return authResponse.Ok, nil
}

func (sl *Client) urlValues() *url.Values {
	uv := url.Values{}
	uv.Add("token", sl.token)
	return &uv
}

func (sl *Client) postMessage(url string, uv *url.Values) ([]byte, error) {
	resp, err := http.PostForm(url, *uv)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(string(t))
	}
	t, _ := ioutil.ReadAll(resp.Body)
	return t, nil
}

func (sl *Client) GetChannels() ([]Channel, error) {
	var uv *url.Values = sl.urlValues()
	t, _ := sl.postMessage(channelsList, uv)
	var chanList ChannelList
	err := json.Unmarshal(t, &chanList)
	if err != nil {
		return nil, err
	}
	return chanList.Channels, nil
}

func (sl *Client) WriteMessage(text string, channel string) error {
	var uv *url.Values = sl.urlValues()
	uv.Add("channel", channel)
	uv.Add("text", text)

	_, err := sl.postMessage(chatPostMessage, uv)
	return err
}