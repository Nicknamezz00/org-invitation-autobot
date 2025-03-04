package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/Nicknamezz00/org-invitation-autobot/store"
	"github.com/Nicknamezz00/org-invitation-autobot/store/generate/model"
	"github.com/Nicknamezz00/org-invitation-autobot/store/generate/query"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const (
	// SENSITIVE environment variables below:
	EnvFeishuAppSecret           = "FEISHU_APP_SECRET"
	EnvGithubPersonalAccessToken = "GITHUB_PERSONAL_ACCESS_TOKEN"

	InvitationStatusPending   = "PENDING"
	InvitationStatusSucceeded = "SUCCEEDED"
	InvitationStatusFailed    = "FAILED"
)

var (
	githubPersonalAccessToken string
)

var lazyInit = map[string]any{
	EnvFeishuAppSecret:           &feishuAppSecret,
	EnvGithubPersonalAccessToken: &githubPersonalAccessToken,
}

func init() {
	if err := MustGetEnvs(); err != nil {
		logrus.Fatalln(err)
	}
	logrus.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint: true,
	})

	viper.SetConfigFile("config/config.yaml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("env")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // e.g. app.port -> APP_PORT
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatalln(err)
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
		//if r := recover(); r != nil {
		//	http.Error(w, fmt.Sprintf("%v", r), http.StatusInternalServerError)
		//}
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
		err = fmt.Errorf("sheetRangeContent error, err=%w", err, contents)
		return
	}

	for _, content := range contents {
		orderID := content.OrderID
		githubName := content.GithubUsername
		githubEmail := content.GithubEmail
		if inviteErr := InviteWrapper(r.Context(), orderID, githubName, githubEmail); inviteErr != nil {
			logrus.WithError(inviteErr).WithFields(logrus.Fields{
				"orderID":     orderID,
				"githubName":  githubName,
				"githubEmail": githubEmail,
			}).Error("invite_error")
		}
	}

}

func main() {
	db := store.New(viper.GetViper())
	query.SetDefault(db)

	c := cron.New()
	c.AddFunc("0 9 * * *", func() { callInviteEndpoint() })
	c.AddFunc("0 21 * * *", func() { callInviteEndpoint() })
	c.Start()

	mux := http.NewServeMux()
	mux.HandleFunc("/invite", invite)
	logrus.Println("HTTP Server listening :8182")
	logrus.Fatalln(http.ListenAndServe(":8182", mux))
}

func InviteWrapper(ctx context.Context, orderID int64, username, email string) (err error) {
	create := &model.InvitationModel{
		ID:               uuid.New().String(),
		OrderID:          orderID,
		GithubUsername:   username,
		GithubEmail:      email,
		InvitationStatus: InvitationStatusPending,
	}
	old, err := query.InvitationModel.WithContext(ctx).Or(
		query.InvitationModel.GithubUsername.Eq(username),
		query.InvitationModel.GithubEmail.Eq(email)).
		First()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("find_old_record_error||err=%v", err)
	}

	if old == nil {
		if err = query.InvitationModel.WithContext(ctx).Create(create); err != nil {
			return fmt.Errorf("create_error||err=%v||create=%+v", err, create)
		}
	} else {
		create.ID = old.ID
		create.OrderID = old.OrderID
		create.GithubUsername = old.GithubUsername
		create.GithubEmail = old.GithubEmail
		create.InvitationStatus = old.InvitationStatus
		create.FirstError = old.FirstError
	}

	defer func() {
		var (
			cause  string
			status = InvitationStatusSucceeded
		)
		if err != nil {
			cause = err.Error()
			status = InvitationStatusFailed
			if errors.Unwrap(err) != nil {
				cause = errors.Unwrap(err).Error()
			}
		}
		if _, err2 := query.InvitationModel.WithContext(ctx).
			Where(query.InvitationModel.ID.Eq(create.ID)).
			UpdateColumnSimple(
				query.InvitationModel.InvitationStatus.Value(status),
				query.InvitationModel.FirstError.Value(cause),
			); err2 != nil {
			logrus.WithField("create", create).WithError(err2).Error("_db_create_error")
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

func callInviteEndpoint() {
	body := strings.NewReader(`{"start":"A2","end":"C"}`)
	resp, err := http.Post("http://localhost:8182/invite", "application/json", body)
	if err != nil {
		logrus.WithError(err).Error("failed to call invite endpoint")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
