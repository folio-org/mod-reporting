package main

import "context"
import "io"
import "strings"
import "time"
import "fmt"
import "regexp"
import "net/http"
import "encoding/json"
import "github.com/jackc/pgx/v5"

// Determine whether this is a MetaDB database, as opposed to LDP Classic
func isMetaDB(dbConn PgxIface) (bool, error) {
	var val int
	magicQuery := "SELECT 1 FROM pg_class c JOIN pg_namespace n ON c.relnamespace=n.oid " +
		"WHERE n.nspname='dbsystem' AND c.relname='main';"
	err := dbConn.QueryRow(context.Background(), magicQuery).Scan(&val)
	if err != nil && strings.Contains(err.Error(), "no rows") {
		// Weirdly, metadb.base_table does not exist on MetaDB
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("could not run isMetaDb query '%s': %w", magicQuery, err)
	}

	return false, nil
}

type dbTable struct {
	SchemaName string `db:"schema_name" json:"tableSchema"`
	TableName  string `db:"table_name" json:"tableName"`
}

type dbColumn struct {
	ColumnName      string `db:"column_name" json:"columnName"`
	DataType        string `db:"data_type" json:"data_type"`
	TableSchema     string `db:"table_schema" json:"tableSchema"`
	TableName       string `db:"table_name" json:"tableName"`
	OrdinalPosition string `db:"ordinal_position" json:"ordinalPosition"`
}

func handleTables(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}
	tables, err := fetchTables(dbConn, session.isMDB)
	if err != nil {
		return fmt.Errorf("could not fetch tables from reporting DB: %w", err)
	}

	return sendJSON(w, tables, "tables")
}

func fetchTables(dbConn PgxIface, isMetaDB bool) ([]dbTable, error) {
	var query string
	if isMetaDB {
		query = `SELECT schema_name, table_name FROM metadb.base_table
			 UNION
			 SELECT 'folio_derived', table_name
			     FROM metadb.table_update t
			         JOIN pg_class c ON c.relname=t.table_name
			         JOIN pg_namespace n ON n.oid=c.relnamespace AND n.nspname=t.schema_name
			     WHERE schema_name='folio_derived'`
	} else {
		query = "SELECT table_name, table_schema as schema_name FROM information_schema.tables WHERE table_schema IN ('local', 'public', 'folio_reporting')"
	}

	rows, err := dbConn.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("could not run query '%s': %w", query, err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[dbTable])
}

func handleColumns(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	v := req.URL.Query()
	schema := v.Get("schema")
	table := v.Get("table")
	if schema == "" || table == "" {
		return fmt.Errorf("must specify both schema and table")
	}

	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}
	columns, err := fetchColumns(dbConn, schema, table)
	if err != nil {
		return fmt.Errorf("could not fetch columns from reporting DB: %w", err)
	}

	return sendJSON(w, columns, "columns")
}

func fetchColumns(dbConn PgxIface, schema string, table string) ([]dbColumn, error) {
	// This seems to work for both MetaDB and LDP Classic
	cols := "column_name, data_type, ordinal_position, table_schema, table_name"
	query := "SELECT " + cols + " FROM information_schema.columns " +
		"WHERE table_schema = $1 AND table_name = $2 AND column_name != $3"
	rows, err := dbConn.Query(context.Background(), query, schema, table, "data")
	if err != nil {
		return nil, fmt.Errorf("could not run query '%s': %w", query, err)
	}
	defer rows.Close()
	/*
		for rows.Next() {
			val, _ := rows.Values()
			fmt.Printf("column 3: %T, %+v\n", val[2], val[2])
		}
	*/

	return pgx.CollectRows(rows, pgx.RowToStructByName[dbColumn])
}

type queryFilter struct {
	Key   string `json:"key"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

type queryOrder struct {
	Key       string `json:"key"`
	Direction string `json:"direction"`
	Nulls     string `json:"nulls"`
}

type queryTable struct {
	Schema  string        `json:"schema"`
	Table   string        `json:"tableName"`
	Filters []queryFilter `json:"columnFilters"`
	Columns []string      `json:"showColumns"`
	Order   []queryOrder  `json:"orderBy"`
	Limit   int           `json:"limit"`
}

type jsonQuery struct {
	Tables []queryTable `json:"tables"`
}

func handleQuery(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}

	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read HTTP request body: %w", err)
	}
	var query jsonQuery
	err = json.Unmarshal(bytes, &query)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON from body: %w", err)
	}

	sql, params, err := makeSql(query)
	if err != nil {
		return fmt.Errorf("could not generate SQL from JSON query: %w", err)
	}

	session.Log("sql", sql, fmt.Sprintf("%v", params))
	rows, err := dbConn.Query(context.Background(), sql, params...)
	if err != nil {
		return fmt.Errorf("could not execute SQL from JSON query: %w", err)
	}

	result, err := collectAndFixRows(rows)
	if err != nil {
		return err
	}

	return sendJSON(w, result, "query result")
}

func makeSql(query jsonQuery) (string, []any, error) {
	if len(query.Tables) != 1 {
		return "", nil, fmt.Errorf("query must have exactly one table")
	}
	qt := query.Tables[0]

	sql := "SELECT " + makeColumns(qt.Columns) + ` FROM "` + qt.Schema + `"."` + qt.Table + `"`
	filterString, params := makeCond(qt.Filters)
	if filterString != "" {
		sql += " WHERE " + filterString
	}
	if len(qt.Order) > 0 {
		sql += " ORDER BY " + makeOrder(qt.Order)
	}
	if qt.Limit != 0 {
		sql += fmt.Sprintf(" LIMIT %d", qt.Limit)
	}

	return sql, params, nil
}

func makeColumns(cols []string) string {
	if len(cols) == 0 {
		return "*"
	}

	s := ""
	for i, col := range cols {
		s += col
		if i < len(cols)-1 {
			s += ", "
		}
	}

	return s
}

func makeCond(filters []queryFilter) (string, []any) {
	params := make([]any, 0)

	s := ""
	for i, filter := range filters {
		if filter.Key == "" {
			continue
		}
		if s != "" {
			s += " AND "
		}
		s += filter.Key
		if filter.Op == "" {
			s += " = "
		} else {
			s += " " + filter.Op + " "
		}
		s += fmt.Sprintf("$%d", i+1)

		params = append(params, filter.Value)
	}

	return s, params
}

func makeOrder(orders []queryOrder) string {
	s := ""
	for i, order := range orders {
		s += order.Key
		s += " " + order.Direction
		// Historically, ui-ldp sends "start" or "end"
		// But we also want to support PostgreSQL's own "FIRST" and "LAST"
		if strings.EqualFold(order.Nulls, "first") ||
			strings.EqualFold(order.Nulls, "start") {
			s += " NULLS FIRST"
		} else {
			s += " NULLS LAST"
		}
		if i < len(orders)-1 {
			s += ", "
		}
	}

	return s
}

type reportQuery struct {
	Url    string            `json:"url"`
	Params map[string]string `json:"params"`
	Limit  int               `json:"limit"`
}

type reportResponse struct {
	TotalRecords int              `json:"totalRecords"`
	Records      []map[string]any `json:"records"`
}

func handleReport(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}

	bytes, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("could not read HTTP request body: %w", err)
	}
	var query reportQuery
	err = json.Unmarshal(bytes, &query)
	if err != nil {
		return fmt.Errorf("could not deserialize JSON from body: %w", err)
	}

	err = validateUrl(query.Url)
	if err != nil {
		return fmt.Errorf("query may not be loaded from %s: %w", query.Url, err)
	}

	resp, err := http.Get(query.Url)
	if err != nil {
		return fmt.Errorf("could not fetch report from %s: %w", query.Url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("could not fetch report from %s: %s", query.Url, resp.Status)
	}

	bytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read report: %w", err)
	}
	sql := string(bytes)

	if session.isMDB && strings.HasPrefix(sql, "--ldp:function") {
		return fmt.Errorf("cannot run LDP Classic report in MetaDB")
	} else if !session.isMDB && strings.HasPrefix(sql, "--metadb:function") {
		return fmt.Errorf("cannot run MetaDB report in LDP Classic")
	}

	if !session.isMDB {
		// LDP Classic needs this, for some reason
		sql = "SET search_path = local, public;\n" + sql
	}

	cmd, params, err := makeFunctionCall(sql, query.Params, query.Limit)
	if err != nil {
		return fmt.Errorf("could not construct SQL function call: %w", err)
	}
	session.Log("sql", cmd, fmt.Sprintf("%v", params))

	tx, err := dbConn.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("could not open transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(), sql)
	if err != nil {
		return fmt.Errorf("could not register SQL function: %w", err)
	}

	rows, err := tx.Query(context.Background(), cmd, params...)
	if err != nil {
		return fmt.Errorf("could not execute SQL from report: %w", err)
	}

	result, err := collectAndFixRows(rows)
	if err != nil {
		return err
	}

	count := len(result) // This is redundant, but it's in the old API so we retain it here
	response := reportResponse{
		TotalRecords: count,
		Records:      result,
	}

	return sendJSON(w, response, "report result")
}

type dbLogEntry struct {
	LogTime       time.Time `db:"log_time" json:"log_time"`
	ErrorSeverity string    `db:"error_severity" json:"error_severity"`
	Message       string    `db:"message" json:"message"`
}

func handleLogs(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	if !session.isMDB {
		return &HTTPError{ http.StatusNotImplemented, "Implemented only for MetaDB, not LDP" }
	}

	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}

	rows, err := dbConn.Query(context.Background(), "SELECT log_time, error_severity, message FROM metadb.log")
	if err != nil {
		return fmt.Errorf("could not fetch logs from reporting DB: %w", err)
	}
	defer rows.Close()

	logs, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbLogEntry])
	if err != nil {
		return fmt.Errorf("could not gather rows of logs from reporting DB: %w", err)
	}

	return sendJSON(w, logs, "logs")
}

type dbVersion struct {
	RawVersion string `db:"mdbversion"`
}

type dbVersionForJson struct {
	RawVersion string `json:"rawVersion"`
	Version    string `json:"version"`
}

func handleVersion(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	if !session.isMDB {
		return &HTTPError{ http.StatusNotImplemented, "Implemented only for MetaDB, not LDP" }
	}

	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}

	rows, err := dbConn.Query(context.Background(), "SELECT mdbversion()")
	if err != nil {
		return fmt.Errorf("could not fetch version from reporting DB: %w", err)
	}
	defer rows.Close()

	versions, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbVersion])
	if err != nil {
		return fmt.Errorf("could not gather rows of version from reporting DB: %w", err)
	}

	row := versions[0]
	jsonRow := dbVersionForJson{row.RawVersion, ""}
	jsonRow.Version = strings.Replace(row.RawVersion, "Metadb v", "", 1)
	return sendJSON(w, jsonRow, "version")
}

type dbUpdate struct {
	TableSchema     string    `db:"schema_name" json:"tableSchema"`
	TableName       string    `db:"table_name" json:"tableName"`
	LastUpdate      time.Time `db:"last_update" json:"lastUpdate"`
	ElapsedRealTime float32   `db:"elapsed_real_time" json:"elapsedRealTime"`
}

func handleUpdates(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	if !session.isMDB {
		return &HTTPError{ http.StatusNotImplemented, "Implemented only for MetaDB, not LDP" }
	}

	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}

	rows, err := dbConn.Query(context.Background(), "SELECT schema_name, table_name, last_update, elapsed_real_time FROM metadb.table_update ORDER BY elapsed_real_time DESC")
	if err != nil {
		return fmt.Errorf("could not fetch updates from reporting DB: %w", err)
	}
	defer rows.Close()

	updates, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbUpdate])
	if err != nil {
		return fmt.Errorf("could not gather rows of updates from reporting DB: %w", err)
	}

	return sendJSON(w, updates, "updates")
}

type dbProcesses struct {
	DBName   string `db:"dbname" json:"databaseName"`
	UserName string `db:"username" json:"userName"`
	State    string `db:"state" json:"state"`
	RealTime string `db:"realtime" json:"realTime"`
	Query    string `db:"query" json:"query"`
}

func handleProcesses(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error {
	if !session.isMDB {
		return &HTTPError{ http.StatusNotImplemented, "Implemented only for MetaDB, not LDP" }
	}

	dbConn, err := session.findDbConn(req.Header.Get("X-Okapi-Token"))
	if err != nil {
		return fmt.Errorf("could not find reporting DB: %w", err)
	}

	rows, err := dbConn.Query(context.Background(), "SELECT dbname, username, state, realtime, query FROM ps() ORDER BY realtime DESC")
	if err != nil {
		return fmt.Errorf("could not fetch processes from reporting DB: %w", err)
	}
	defer rows.Close()

	processes, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbProcesses])
	if err != nil {
		return fmt.Errorf("could not gather rows of processes from reporting DB: %w", err)
	}

	return sendJSON(w, processes, "processes")
}

func validateUrl(_url string) error {
	// We could sanitize the URL, rejecting requests using unauthorized sources: see issue #36
	return nil
}

func makeFunctionCall(sql string, params map[string]string, limit int) (string, []any, error) {
	orderedParams := make([]any, 0)

	re := regexp.MustCompile(`--.+:function\s+(.+)`)
	m := re.FindStringSubmatch(sql)
	if m == nil {
		return "", nil, fmt.Errorf("could not extract SQL function name")
	}

	s := make([]string, 0, len(params))
	i := 1
	for key, val := range params {
		s = append(s, fmt.Sprintf("%s => $%d", key, i))
		orderedParams = append(orderedParams, val)
		i++
	}

	cmd := "SELECT * FROM " + m[1] + "(" + strings.Join(s, ", ") + ")"
	if limit != 0 {
		cmd += fmt.Sprintf(" LIMIT %d", limit)
	}

	return cmd, orderedParams, nil
}

func collectAndFixRows(rows pgx.Rows) ([]map[string]any, error) {
	records, err := pgx.CollectRows(rows, pgx.RowToMap)
	// fmt.Printf("rows: %+v\n", rows.FieldDescriptions())
	if err != nil {
		return nil, fmt.Errorf("could not collect query result data: %w", err)
	}

	// Fix up types
	for _, rec := range records {
		for key, val := range rec {
			switch v := val.(type) {
			case [16]uint8:
				// This is how pgx represents fields of type "uuid"
				rec[key] = fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:16])
			default:
				// Nothing to do
			}
		}
	}

	return records, nil
}

func sendJSON(w http.ResponseWriter, data any, caption string) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not encode JSON for %s: %w", caption, err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bytes)
	return err
}
