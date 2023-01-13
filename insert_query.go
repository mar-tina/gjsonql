package gjsonql

import (
	"database/sql"
	"strings"

	"github.com/tidwall/gjson"
)

var insert_tmpl = `INSERT INTO {{.Tbl}} ({{.Columns}})  VALUES ({{range $idx, $val := .ColumnsArr}}{{if eq $idx 0}} {{else}},{{end}}?{{end}})` + ";"

type InsertQuery struct {
	QueryKind string
	QueryName string
	Tbl       *InsertTbl
}

type InsertTbl struct {
	TblsArr    []string
	Tbl        string
	Columns    string
	Values     string
	ColumnsArr []string
	ValuesArr  []string
	TmplStr    string
}

func Insert(data string) *InsertQuery {
	iquery := &InsertQuery{
		QueryKind: "insertQuery",
	}
	tbl := &InsertTbl{}
	gjson.Get(data, "@this").ForEach(func(tblname, columns gjson.Result) bool {
		tbl.Tbl = tblname.Str
		tbl.handleColumnValuesIsolate(columns.Raw)
		return true
	})

	tbl.Columns = strings.Join(tbl.ColumnsArr, ",")
	tbl.TmplStr = TmplExecute(insert_tmpl, tbl)
	iquery.Tbl = tbl
	return iquery
}

func (tbl *InsertTbl) handleColumnValuesIsolate(data string) {
	gjson.Get(data, "@this").ForEach(func(tbl_col, value gjson.Result) bool {
		if value.Type == gjson.String {
			valueStr := value.Str
			clean := strings.ReplaceAll(valueStr, "@", "")
			tbl.ColumnsArr = append(tbl.ColumnsArr, tbl_col.Str)
			tbl.ValuesArr = append(tbl.ValuesArr, clean)
			return true
		}

		tbl.ColumnsArr = append(tbl.ColumnsArr, tbl_col.Str)
		tbl.ValuesArr = append(tbl.ValuesArr, value.Str)
		return true
	})
}

func (iquery *InsertQuery) Execute(db *sql.DB, data map[string]interface{}) map[string][]map[string]interface{} {
	r := make(map[string][]map[string]interface{})
	tbl := iquery.Tbl
	smt, err := db.Prepare(tbl.TmplStr)
	if m := handleErr(err); m != nil {
		r["err"] = append(r["err"], m)
		return r
	}
	input := mapData(tbl.ValuesArr, data)
	_, err = smt.Exec(input...)
	if m := handleErr(err); m != nil {
		r["err"] = append(r["err"], m)
		return r
	}

	r["msg"] = append(r["msg"], map[string]interface{}{
		"response": "completed successfully",
	})
	return r
}

func (iquery *InsertQuery) Kind() string {
	return iquery.QueryKind
}

func (iquery *InsertQuery) Name() string {
	return iquery.QueryName
}
