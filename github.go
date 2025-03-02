package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

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
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("[MUST NOTICE] req=%+v||json=%v||code=%v||resp=%s", req, string(jsonData), resp.StatusCode, string(bytes))
	}
	log.Println(string(bytes))
	return nil
}
