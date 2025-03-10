package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type InviteResponse struct {
	Message string `json:"message"`
	Errors  []struct {
		Resource string `json:"resource"`
		Code     string `json:"code"`
		Field    string `json:"field"`
		Message  string `json:"message"`
	} `json:"errors"`
	DocumentationUrl string `json:"documentation_url"`
	Status           string `json:"status"`
}

func CheckIfUserIsMember(ctx context.Context, username string) (bool, error) {
	url := "https://api.github.com/orgs/Nicknamezz00-organization/members/" + username
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", githubPersonalAccessToken))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	bytes, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusFound:
		return false, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unhandled case||resp=%s||code=%v", string(bytes), resp.StatusCode)
	}
}

var ErrAlreadyInvited = errors.New("already invited, skip")

func Invite(username, email string) error {
	url := "https://api.github.com/orgs/Nicknamezz00-organization/invitations"
	data := map[string]any{
		"email":    email,
		"role":     "direct_member",
		"team_ids": []int64{},
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", githubPersonalAccessToken))
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytes, _ := io.ReadAll(resp.Body)
	var r InviteResponse
	_ = json.Unmarshal(bytes, &r)

	if resp.StatusCode != http.StatusCreated {
		if strings.Contains(r.Errors[0].Message, "already a part of this organization") {
			logrus.Debugf("%s is already a part of this organization", username)
			return ErrAlreadyInvited
		}
		return fmt.Errorf("[MUST NOTICE]||req=%+v||json=%v||code=%v||resp=%s", req, string(jsonData), resp.StatusCode, string(bytes))
	}
	return nil
}
