package gjsonql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

var create_tmpl = `CREATE TABLE IF NOT EXISTS {{.Name}} ({{range $idx, $val := .Columns}} {{if eq $idx 0}} {{.}} {{else}} , {{.}} {{end}} {{end}} ,{{.Primary}});`

type CreateQuery struct {
	QueryKind string
	QueryName string
	Queries   []string
	Tbls      []*Tbl
}

type Tbl struct {
	Name    string
	Columns []string
	Primary string
	TmplStr string
}

func Create(data string) *CreateQuery {
	cquery := &CreateQuery{
		QueryKind: "createQuery",
	}
	gjson.Get(data, "@this").ForEach(func(key, value gjson.Result) bool {
		tbl := handleTable(key.Str, value.Raw)
		out := TmplExecute(create_tmpl, tbl)
		tbl.TmplStr = out
		cquery.Tbls = append(cquery.Tbls, tbl)
		return true
	})
	return cquery
}

func handleTable(tblname string, data string) *Tbl {
	tbl := &Tbl{
		Name: tblname,
	}
	var appendLater []string
	gjson.Get(data, "@this").ForEach(func(colunm_name, properties gjson.Result) bool {
		if len(strings.Split(colunm_name.Str, "_")) < 2 {
			columns := parseProperties(colunm_name.Str, properties.Str)
			if !strings.Contains(columns[0], "PRIMARY KEY") {
				tbl.Columns = append(tbl.Columns, columns...)
				return true
			}
			tbl.Columns = append(tbl.Columns, columns[1])
			tbl.Primary = columns[0]
			return true
		}

		columns := handleForeignKey(colunm_name.Str, properties.Str)
		tbl.Columns = append(tbl.Columns, columns[0])
		appendLater = append(appendLater, columns[1])
		return true
	})

	tbl.Columns = append(tbl.Columns, appendLater...)
	return tbl
}

func handleForeignKey(column_name, properties string) []string {
	columnAsArr := strings.Split(column_name, "_")
	columns := []string{}
	tbl := columnAsArr[0]
	refkey := columnAsArr[1]
	columns = append(columns, fmt.Sprintf("%s %s", column_name, properties))
	columns = append(columns, fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)", column_name, tbl, refkey))
	return columns
}

func parseProperties(column_name, properties string) []string {
	propertiesArr := strings.Split(properties, "|")
	var columns []string
	current := column_name
	for _, prop := range propertiesArr {
		switch prop {
		case "primary":
			columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", column_name))
			continue
		case "integer", "varchar", "not null", "unique":
			current += " " + prop
			continue
		}
	}
	columns = append(columns, current)
	return columns
}

func (cquery *CreateQuery) Execute(db *sql.DB, data map[string]interface{}) map[string][]map[string]interface{} {
	r := make(map[string][]map[string]interface{})
	for _, tbl := range cquery.Tbls {
		smt, err := db.Prepare(tbl.TmplStr)
		if m := handleErr(err); m != nil {
			r["err"] = append(r["err"], m)
			return r
		}
		_, err = smt.Exec()
		if m := handleErr(err); m != nil {
			r["err"] = append(r["err"], m)
			return r
		}
	}

	r["msg"] = append(r["msg"], map[string]interface{}{
		"response": "completed successfully",
	})
	return r
}

func (cquery *CreateQuery) Kind() string {
	return cquery.QueryKind
}

func (cquery *CreateQuery) Name() string {
	return cquery.QueryName
}
