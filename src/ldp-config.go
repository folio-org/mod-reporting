// handle the /ldp/config and /ldp/config/{key} endpoints
package main

import "io"
import "fmt"
import "strings"
import "net/http"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/indexdata/foliogo"


// Types for what we read from mod-settings
// "General" here means not knowing the structure of the value

type settingsItemGeneral struct {
	Id string `json:"id"`
	Scope string `json:"scope"`
	Key string `json:"key"`
	Value interface{}
}

type settingsResultInfoGeneral struct {
	TotalRecords int `json:"totalRecords"`
	// We don't care about diagnostics
}

type settingsResponseGeneral struct {
	Items []settingsItemGeneral `json:"items"`
	ResultInfo settingsResultInfoGeneral `json:"resultInfo"`
}


// Types for what we return as /ldp/config

type configItem struct {
	Key string `json:"key"`
	Tenant string `json:"tenant"`
	Value string `json:"value"`
}


// Wrapper for foliogo.Fetch that extracts token from request and sends it
func fetchWithToken(req *http.Request, folioSession foliogo.Session, path string, params foliogo.RequestParams) ([]byte, error) {
	token := req.Header.Get("X-Okapi-Token")
	if token != "" {
		params.Token = token
	}

	return folioSession.Fetch(path, params)
}


func fetchWithToken0(req *http.Request, folioSession foliogo.Session, path string) ([]byte, error) {
	return fetchWithToken(req, folioSession, path, foliogo.RequestParams{})
}


func settingsItemToConfigItem(item settingsItemGeneral, tenant string) (configItem, error) {
	value, ok := item.Value.(string)
	if !ok {
		// mod-settings can contain values of any type: needs serializing
		bytes, err := json.Marshal(item.Value)
		if err != nil {
			return configItem{}, fmt.Errorf("could not serialize value from mod-settings: %w", err)
		}
		value = string(bytes)
	}
	ci := configItem{
		Key: item.Key,
		Value: value,
		Tenant: tenant,
	}
	return ci, nil
}


// The /ldp/config endpoint only supports GET, with no URL parameters
func handleConfig(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	bytes, err := fetchWithToken0(req, session.folioSession, `settings/entries?query=scope=="ui-ldp.admin"`)
	if err != nil {
		return fmt.Errorf("could not fetch from mod-settings: %w", err)
	}

	var r settingsResponseGeneral
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON from mod-settings: %w", err)
	}

	// XXX in a system with many settings, we might get back less
	// than the full set, in which case we'd need to using paging
	// to accumulate them all. In practice, that's never going to
	// happen. But we could look at resultInfo.totalRecords to
	// determine whether this has happened.

	tenant := session.folioSession.GetTenant()
	config := make([]configItem, len(r.Items))
	for i, item := range(r.Items) {
		config[i], err = settingsItemToConfigItem(item, tenant)
		if err != nil {
			return err
		}
	}

	bytes, err = json.Marshal(config)
	if err != nil {
		return fmt.Errorf("could not serialize JSON: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(bytes)
	return nil
}


// The /ldp/config/{key} endpoint only supports GET and PUT
func handleConfigKey(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	key := strings.Replace(req.URL.Path, "/ldp/config/", "", 1)
	var bytes []byte
	var err error

	if req.Method == "PUT" {
		return writeConfigKey(w, req, session, key)
	}

	// Assume GET
	path := `settings/entries?query=scope=="ui-ldp.admin"+and+key=="` + key + `"`
	bytes, err = fetchWithToken0(req, session.folioSession, path)
	if err != nil {
		return fmt.Errorf("could not read from mod-settings: %w", err)
	}

	var r settingsResponseGeneral
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON %+v from mod-settings: %w", bytes, err)
	}

	if r.ResultInfo.TotalRecords < 1 {
		return fmt.Errorf("no config item with key '%s'", key)
	}

	item := r.Items[0]
	tenant := session.folioSession.GetTenant()
	config, err := settingsItemToConfigItem(item, tenant)
	if err != nil {
		return err
	}

	bytes, err = json.Marshal(config)
	if err != nil {
		return fmt.Errorf("could not serialize JSON: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bytes)
	return err
}


func writeConfigKey(w http.ResponseWriter, req *http.Request, session *ModReportingSession, key string) (error) {
	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read HTTP request body: %w", err)
	}

	var item configItem
	err = json.Unmarshal(bytes, &item)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON from body: %w", err)
	}
	// fmt.Println("item.Value =", item.Value)

	// Irritatingly, the WSAPI for mod-settings is different if
	// we're creating a new key from if we're replacing an
	// existing one, so we need first to search for an existing
	// record
	path := `settings/entries?query=scope=="ui-ldp.admin"+and+key=="` + key + `"`
	bytes, err = fetchWithToken0(req, session.folioSession, path)
	if err != nil {
		return fmt.Errorf("could not read from mod-settings: %w", err)
	}

	var r settingsResponseGeneral
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON %+v from mod-settings: %w", bytes, err)
	}

	var id, method string
	if r.ResultInfo.TotalRecords > 0 {
		// We need to PUT to the existing record
		id = r.Items[0].Id
		method = "PUT"
		path = "settings/entries/" + id
	} else {
		dumbId, err2 := uuid.NewRandom()
		if err2 != nil {
			return fmt.Errorf("could not generate v4 UUID: %w", err2)
		}
		id = dumbId.String()
		method = "POST"
		path = "settings/entries"
	}

	var simpleSettingsItem map[string]interface{} = map[string]interface{}{
		"id": id,
		"scope": "ui-ldp.admin",
		"key": key,
		"value": item.Value,
	}
	_, err = fetchWithToken(req, session.folioSession, path, foliogo.RequestParams{
		Method: method,
		Json: simpleSettingsItem,
	})
	if err != nil {
		return fmt.Errorf("could not write to mod-settings: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	bytes, err = json.Marshal(simpleSettingsItem)
	if err != nil {
		return fmt.Errorf("could not serialize JSON for response: %w", err)
	}
	_, err = w.Write(bytes)
	return err
}
