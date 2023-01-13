package gjsonql

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/tidwall/gjson"
)

var update_tmpl = `UPDATE {{.Tbl}} SET {{range .Columns}}{{.}} {{end}}{{if not .WhereArr}}{{else}}WHERE {{range .WhereArr}} {{.}} {{end}}{{end}};`

type UpdateQuery struct {
	QueryKind string
	QueryName string
	Tbl       *UpdateTbl
}

type UpdateTbl struct {
	Tbl            string
	Columns        []string
	ColumnKeys     []string
	WhereArr       []string
	WhereValueKeys []string
	TmplStr        string
}

func Update(data string) *UpdateQuery {
	uquery := &UpdateQuery{
		QueryKind: "updateQuery",
	}
	gjson.Get(data, "@this").ForEach(func(tbl, columns gjson.Result) bool {
		if columns.Type == gjson.JSON {
			uquery.handleTableUpdate(false, tbl.Str, columns.Raw)
		}
		return true
	})
	uquery.Tbl.TmplStr = TmplExecute(update_tmpl, uquery.Tbl)
	log.Printf("TMP: %v", uquery.Tbl.TmplStr)
	uquery.Tbl.ColumnKeys = append(uquery.Tbl.ColumnKeys, uquery.Tbl.WhereValueKeys...)
	return uquery
}

func (uquery *UpdateQuery) handleTableUpdate(isArray bool, tblname, columns string) {
	tbl := &UpdateTbl{
		Tbl: tblname,
	}

	gjson.Get(columns, "@this").ForEach(func(column_key, column_value gjson.Result) bool {
		if column_value.Type == gjson.JSON {
			if strings.Contains(column_key.Str, "#") {
				name := strings.Split(replaceAllDirty(column_key.Str), "_")[0]
				tblname := strings.Split(replaceAllDirty(column_key.Str), "_")[1]
				innertbl := &SelectTbl{
					Name:  tblname,
					Alias: replaceAllDirty(column_key.Str),
					Kind:  getTblKind(column_key.Str),
				}
				innertbl.Parse(column_value.Raw)
				innertbl.TmplStr = TmplExecute(select_tmpl, innertbl)
				innertbl.InnerTmpl = fmt.Sprintf("%s %s IN (%s)", getSeparator(column_key.Str), name, innertbl.TmplStr)
				tbl.WhereValueKeys = append(tbl.WhereValueKeys, innertbl.Wherekeys...)
				tbl.WhereArr = append(tbl.WhereArr, innertbl.InnerTmpl)
				return true
			}
		}
		if strings.Contains(column_value.Str, "@") {
			query, state := buildUpdateWhereClause(tblname, column_value.Str, column_key.Str)
			tbl.WhereArr = append(tbl.WhereArr, query)
			tbl.WhereValueKeys = append(tbl.WhereValueKeys, state)
			return true
		}
		tbl.Columns = append(tbl.Columns, column_key.Str+" "+"="+" "+"?")
		tbl.ColumnKeys = append(tbl.ColumnKeys, column_value.Str)
		return true
	})

	uquery.Tbl = tbl
}

func buildUpdateWhereClause(tblname, column_value, column_key string) (string, string) {
	if strings.Contains(column_value, "#") {
		searchAndState := strings.Split(column_value, "@")
		search := searchAndState[0]
		state := searchAndState[1]
		clean := replaceAllDirty(search)
		log.Printf("sss: %v %v", search, search)
		query := fmt.Sprintf("EXISTS (SELECT %s FROM %s WHERE %s = ?)", column_key, tblname, clean)
		return query, state
	}
	clean := replaceAllDirty(column_value)
	query := getSeparator(column_value) + " " + column_key + getOperator(column_value) + " " + "?"
	// fmt.Sprintf(" %s %s = ?", getSeparator(column_value), " ", column_key)
	return query, clean
}

func (uquery *UpdateQuery) Kind() string {
	return uquery.QueryKind
}

func (uquery *UpdateQuery) Name() string {
	return uquery.QueryName
}

func (uquery *UpdateQuery) Execute(db *sql.DB, data map[string]interface{}) map[string][]map[string]interface{} {
	r := make(map[string][]map[string]interface{})
	tbl := uquery.Tbl
	smt, err := db.Prepare(tbl.TmplStr)
	if m := handleErr(err); m != nil {
		r["err"] = append(r["err"], m)
		return r
	}
	input := mapData(tbl.ColumnKeys, data)
	_, err = smt.Exec(input...)
	if m := handleErr(err); m != nil {
		r["err"] = append(r["err"], m)
		return r
	}

	r["msg"] = append(r["msg"], map[string]interface{}{
		"response": "completed successfully",
	})
	return nil
}
