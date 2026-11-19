package reporting

import "os"
import "errors"
import "fmt"
import "encoding/json"
import "github.com/indexdata/foliogo"

type settingsValue struct {
	Url  string `json:"url"`
	Pass string `json:"pass"`
	User string `json:"user"`
}

// The value may be either a JSON object or a string containing an
// encoded JSON object (see issue #60), so we decode it in two stages.
type settingsItem struct {
	Value json.RawMessage `json:"value"`
}

type settingsResultInfo struct {
	TotalRecords int `json:"totalRecords"`
}

type settingsResponse struct {
	Items      []settingsItem     `json:"items"`
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
	bytes, err := session.Fetch("settings/entries?query=scope==%22ui-ldp.admin%22+and+key==%22dbinfo%22", params)
	if err != nil {
		return "", "", "", fmt.Errorf("cannot fetch 'dbinfo' from config: %w", err)
	}

	value, err := parseDbInfo(bytes)
	if err != nil {
		return "", "", "", err
	}
	return value.Url, value.User, value.Pass, nil
}

func parseDbInfo(bytes []byte) (settingsValue, error) {
	var value settingsValue

	var r settingsResponse
	err := json.Unmarshal(bytes, &r)
	if err != nil {
		return value, fmt.Errorf("decode 'dbinfo' JSON failed: %w", err)
	}

	if r.ResultInfo.TotalRecords < 1 || len(r.Items) < 1 {
		return value, errors.New("no 'dbinfo' setting in FOLIO database")
	}

	raw := r.Items[0].Value
	if len(raw) > 0 && raw[0] == '"' {
		var encoded string
		err = json.Unmarshal(raw, &encoded)
		if err != nil {
			return value, fmt.Errorf("decode 'dbinfo' string value failed: %w", err)
		}
		raw = []byte(encoded)
	}

	err = json.Unmarshal(raw, &value)
	if err != nil {
		return value, fmt.Errorf("decode 'dbinfo' value failed: %w", err)
	}
	return value, nil
}
