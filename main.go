package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/Nicknamezz00/org-invitation-autobot/store/generate/model"
	"github.com/Nicknamezz00/org-invitation-autobot/store/generate/query"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	// SENSITIVE environment variables below:
	EnvFeishuUserAccessToken     = "FEISHU_USER_ACCESS_TOKEN"
	EnvFeishuAppSecret           = "FEISHU_APP_SECRET"
	EnvGithubPersonalAccessToken = "GITHUB_PERSONAL_ACCESS_TOKEN"

	InvitationStatusSucceeded = "SUCCEEDED"
	InvitationStatusFailed    = "FAILED"
)

var (
	githubPersonalAccessToken string
)

var lazyInit = map[string]any{
	EnvFeishuUserAccessToken:     &feishuUserAccessToken,
	EnvFeishuAppSecret:           &feishuAppSecret,
	EnvGithubPersonalAccessToken: &githubPersonalAccessToken,
}

func init() {
	if err := MustGetEnvs(); err != nil {
		log.Fatalln(err)
	}

	viper.SetConfigFile("config/config.yaml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("env")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // e.g. app.port -> APP_PORT
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}
}

func invite(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		statusCode = http.StatusOK
		rng        struct {
			Start string `json:"start"`
			End   string `json:"end"`
		}
	)
	defer func() {
		if err != nil {
			http.Error(w, err.Error(), statusCode)
		}
	}()

	if r.Method != http.MethodPost {
		err = errors.New("method not allowed")
		statusCode = http.StatusMethodNotAllowed
		return
	}

	if err = json.NewDecoder(r.Body).Decode(&rng); err != nil {
		statusCode = http.StatusBadRequest
		err = fmt.Errorf("bind request error, err=%w", err)
		return
	}
	if rng.Start == "" || rng.End == "" {
		statusCode = http.StatusBadRequest
		err = fmt.Errorf("invalid params, range=%+v||err=%w", rng, err)
		return
	}

	contents, err := SheetRangeContent(rng.Start, rng.End)
	if err != nil {
		statusCode = http.StatusOK
		err = fmt.Errorf("sheetRangeContent error, err=%w", err)
		return
	}

	for _, content := range contents {
		orderID := cast.ToInt64(content[0])
		githubName := cast.ToString(content[1])
		githubEmail := cast.ToString(content[2])
		if inviteErr := InviteWrapper(orderID, githubName, githubEmail); inviteErr != nil {
			log.Printf("invite_error||err=%v||orderID=%d||githubName=%s||githubEmail=%s\n", err, orderID, githubName, githubEmail)
		}
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/invite", invite)
	log.Println("HTTP Server listening :8182")
	log.Fatal(http.ListenAndServe(":8182", mux))
}

func InviteWrapper(orderID int64, username, email string) (err error) {
	defer func() {
		status := InvitationStatusSucceeded
		if err != nil {
			status = InvitationStatusFailed
		}

		create := model.InvitationModel{
			ID:               uuid.New().String(),
			OrderID:          orderID,
			GithubUsername:   username,
			GithubEmail:      email,
			InvitationStatus: status,
			FirstError:       err.Error(),
		}
		if dbErr := query.InvitationModel.WithContext(context.Background()).Create(&create); dbErr != nil {
			log.Println("DB Error||err=%v||create=%+v", err, create)
		}
	}()

	if !purchase(orderID) {
		return fmt.Errorf("not purchased||orderID=%d||name=%s||email=%s")
	}
	return Invite(username, email)
}

func MustGetEnvs() (err error) {
	for key := range lazyInit {
		if value, exist := os.LookupEnv(key); !exist || value == "" {
			return fmt.Errorf("env '%s' not exist", key)
		}
	}
	for key, ptr := range lazyInit {
		if value, exist := os.LookupEnv(key); exist {
			reflect.ValueOf(ptr).Elem().SetString(value)
		}
	}
	return nil
}
