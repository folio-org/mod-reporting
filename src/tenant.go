package main

import "io"
import "fmt"
import "net/http"
import "encoding/json"
import "github.com/indexdata/foliogo"


type jsonTenantParameters struct {
	Key string `json:"key"`
	Value string `json:"value"`
}

type jsonTenantApi struct {
	Module_to string `json:"module_to"`
	Parameters []jsonTenantParameters `json:parameters`
}

func handleTenantAPI(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read HTTP request body: %w", err)
	}

	var tenantApi jsonTenantApi
	err = json.Unmarshal(bytes, &tenantApi)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON from body: %w", err)
	}
	session.Log("tenant", "module_to:", tenantApi.Module_to, "-- parameters:", fmt.Sprintf("%+v", tenantApi.Parameters))
	for _, param := range tenantApi.Parameters {
		if param.Key == "loadSample" && param.Value == "true" {
			return loadSample(w, req, session)
		}
	}

	return nil
}

func loadSample(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	id := "bd76ccec-2942-41f2-9bde-38f562d41842"
	 body := fmt.Sprintf(`{
  "id": "%s",
  "scope": "ui-ldp.admin",
  "key": "tqrepos",
  "value": [
    {
      "type": "gitlab",
      "user": "MikeTaylor",
      "repo": "metadb-queries",
      "branch": "main",
      "dir": "folio/reports"
    },
    {
      "user": "metadb-project",
      "repo": "metadb-examples",
      "branch": "main",
      "dir": "folio/reports"
    }
  ]
}`, id)
	session.Log("tenant", "loading sample data")
	_, err := fetchWithToken(req, session.folioSession, "settings/entries", foliogo.RequestParams{
		Method: "POST",
		Body:   body,
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("could not POST to mod-settings: %w", err)
	}

	// XXX should allow for 304 if the ID already exists

	return nil;
}
