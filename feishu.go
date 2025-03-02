package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larksheets "github.com/larksuite/oapi-sdk-go/v3/service/sheets/v3"
)

const (
	feishuSpreadsheet = `Xhs2sax3GhvF3rt7atLcjxl1nwd`
	feishuAppID       = "cli_a749705063fa100c"
)

var (
	feishuUserAccessToken string
	feishuAppSecret       string
)

// SDK 使用文档：https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/server-side-sdk/golang-sdk-guide/preparations
// 复制该 Demo 后, 需要将 "YOUR_APP_ID", "YOUR_APP_SECRET" 替换为自己应用的 APP_ID, APP_SECRET.
// 以下示例代码默认根据文档示例值填充，如果存在代码问题，请在 API 调试台填上相关必要参数后再复制代码使用
func GetSheets() (*larksheets.QuerySpreadsheetSheetResp, error) {
	client := lark.NewClient(feishuAppID, feishuAppSecret)
	req := larksheets.NewQuerySpreadsheetSheetReqBuilder().
		SpreadsheetToken(feishuSpreadsheet).
		Build()
	opts := larkcore.WithUserAccessToken(feishuUserAccessToken)

	resp, err := client.Sheets.V3.SpreadsheetSheet.Query(context.Background(), req, opts)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if !resp.Success() {
		return nil, fmt.Errorf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	log.Println(larkcore.Prettify(resp))
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

func SheetRangeContent(start, end string) ([][]any, error) {
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", feishuUserAccessToken))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	bytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	log.Println(larkcore.Prettify(string(bytes)))

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

	return apiResponse.Data.ValueRange.Values, nil
}
