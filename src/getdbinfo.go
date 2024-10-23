package main

import "os"
import "errors"
import "strings"
import "fmt"
import "encoding/json"
import "github.com/indexdata/foliogo"

type settingsValue struct {
	Url  string `json:"url"`
	Pass string `json:"pass"`
	User string `json:"user"`
}

type settingsItem struct {
	Value settingsValue `json:"value"`
}

type settingsResultInfo struct {
	TotalRecords int `json:"totalRecords"`
}

type settingsResponse struct {
	Items      []settingsItem     `json:"items"`
	ResultInfo settingsResultInfo `json:"resultInfo"`
}

// If the value is encoded as a string: see issue #60
type oldSettingsItem struct {
	Value string `json:"value"`
}

type oldSettingsResponse struct {
	Items      []oldSettingsItem  `json:"items"`
	ResultInfo settingsResultInfo `json:"resultInfo"`
}

func getDbInfo(session foliogo.Session, token string) (string, string, string, error) {
	// If defined, environment variables override the setting from the database
	dburl := os.Getenv("REPORTING_DB_URL")
	dbuser := os.Getenv("REPORTING_DB_USER")
	dbpass := os.Getenv("REPORTING_DB_PASS")
	if dburl != "" && dbuser != "" && dbpass != "" {
		return dburl, dbuser, dbpass, nil
	}

	params := foliogo.RequestParams{Token: token}
	bytes, err := session.Fetch(`settings/entries?query=scope=="ui-ldp.admin"+and+key=="dbinfo"`, params)
	if err != nil {
		return "", "", "", fmt.Errorf("cannot fetch 'dbinfo' from config: : %w", err)
	}

	var r settingsResponse
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		if !strings.Contains(err.Error(), "Go struct field settingsItem.items.value") {
			return "", "", "", fmt.Errorf("decode 'dbinfo' JSON failed: %w", err)
		}

		var oldR oldSettingsResponse
		err = json.Unmarshal(bytes, &oldR)
		if err != nil {
			return "", "", "", fmt.Errorf("decode 'dbinfo' old-style JSON failed: %w", err)
		}

		err = convertResultInfo(oldR, &r)
		if err != nil {
			return "", "", "", err
		}
	}

	if r.ResultInfo.TotalRecords < 1 {
		return "", "", "", errors.New("no 'dbinfo' setting in FOLIO database")
	}
	value := r.Items[0].Value
	return value.Url, value.User, value.Pass, nil
}

func convertResultInfo(oldR oldSettingsResponse, r *settingsResponse) error {
	r.ResultInfo = oldR.ResultInfo

	count := len(oldR.Items)
	r.Items = make([]settingsItem, count)
	for i := 0; i < count; i++ {
		err := json.Unmarshal([]byte(oldR.Items[i].Value), &r.Items[i].Value)
		if err != nil {
			return fmt.Errorf("decode 'dbinfo' old-style value failed: %w", err)
		}
	}

	return nil
}
