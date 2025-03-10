package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larksheets "github.com/larksuite/oapi-sdk-go/v3/service/sheets/v3"
	"github.com/spf13/cast"
)

const (
	feishuSpreadsheet = `Xhs2sax3GhvF3rt7atLcjxl1nwd`
	feishuAppID       = "cli_a749705063fa100c"
)

var (
	feishuAppSecret string
)

func acquireFeishuTenantAccessToken() (string, error) {
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	payload, err := json.Marshal(map[string]any{
		"app_id":     feishuAppID,
		"app_secret": feishuAppSecret,
	})
	if err != nil {
		return "", fmt.Errorf("marshal payload error||err=%w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return "", fmt.Errorf("new request error||err=%w", err)
	}
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	req.Header.Add("Host", "open.feishu.cn")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request error||err=%w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read body error||err=%w", err)
	}

	type APIResponse struct {
		Code              int    `json:"code"`
		Expire            int    `json:"expire"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	var resp APIResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("bind response error||resp=%s||err=%w", string(body), err)
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("response code non-zero||resp=%s||err=%w", string(body), err)
	}
	return resp.TenantAccessToken, nil
}

// SDK 使用文档：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/server-side-sdk/golang-sdk-guide/preparations
// 复制该 Demo 后, 需要将 "YOUR_APP_ID", "YOUR_APP_SECRET" 替换为自己应用的 APP_ID, APP_SECRET.
// 以下示例代码默认根据文档示例值填充，如果存在代码问题，请在 API 调试台填上相关必要参数后再复制代码使用
func GetSheets() (*larksheets.QuerySpreadsheetSheetResp, error) {
	client := lark.NewClient(feishuAppID, feishuAppSecret)
	req := larksheets.NewQuerySpreadsheetSheetReqBuilder().
		SpreadsheetToken(feishuSpreadsheet).
		Build()

	feishuTenantAccessToken, err := acquireFeishuTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("error acquiring tenant access token||err=%w", err)
	}
	opts := larkcore.WithTenantAccessToken(feishuTenantAccessToken)
	resp, err := client.Sheets.V3.SpreadsheetSheet.Query(context.Background(), req, opts)
	if err != nil {
		return nil, err
	}
	if !resp.Success() {
		return nil, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	return resp, nil
}

func FirstSheet(resp *larksheets.QuerySpreadsheetSheetResp) (string, error) {
	if len(resp.Data.Sheets) == 0 {
		return "", errors.New("no sheets")
	}
	if resp.Data.Sheets[0].SheetId == nil {
		return "", errors.New("sheetID is nil")
	}
	return *resp.Data.Sheets[0].SheetId, nil
}

type Range struct {
	OrderID        int64
	GithubUsername string
	GithubEmail    string
}

func SheetRangeContent(start, end string) ([]Range, error) {
	sheets, err := GetSheets()
	if err != nil {
		return nil, err
	}
	sheetID, err := FirstSheet(sheets)
	if err != nil {
		return nil, err
	}
	queryRange := fmt.Sprintf(`%s!%s:%s`, sheetID, start, end)
	fullURL := fmt.Sprintf("https://open.feishu.cn/open-apis/sheets/v2/spreadsheets/%s/values/%s", feishuSpreadsheet, queryRange)
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	feishuTenantAccessToken, err := acquireFeishuTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("error acquiring tenant access token||err=%w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", feishuTenantAccessToken))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	bytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	type APIResponse struct {
		Data struct {
			ValueRange struct {
				Values [][]any `json:"values"`
			} `json:"valueRange"`
		} `json:"data"`
	}
	var apiResponse APIResponse
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		return nil, err
	}

	return parseContent(apiResponse.Data.ValueRange.Values)
}

func parseContent(values [][]any) (r []Range, err error) {
	for _, v := range values {
		data := Range{
			OrderID:        cast.ToInt64(v[0]),
			GithubUsername: cast.ToString(v[1]),
		}
		if _, err = cast.ToStringE(v[2]); err == nil {
			data.GithubEmail = cast.ToString(v[2])
		} else {
			type CellValue struct {
				Link string `json:"link"`
				Text string `json:"text"`
				Type string `json:"type"`
			}
			cellData, ok := v[2].([]interface{})
			if !ok || len(cellData) == 0 {
				return nil, fmt.Errorf("invalid cell data format")
			}
			cellMap, ok := cellData[0].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid cell map format")
			}
			cellBytes, err := json.Marshal(cellMap)
			if err != nil {
				return nil, fmt.Errorf("marshal cell map error, err=%v", err)
			}
			var cell CellValue
			if err = json.Unmarshal(cellBytes, &cell); err != nil {
				return nil, fmt.Errorf("unmarshal cell error, err=%v", err)
			}
			data.GithubEmail = cell.Text
		}
		r = append(r, data)
	}
	return r, nil
}
