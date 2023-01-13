package gjsonql

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/tidwall/gjson"
)

var limit_tmpl = `{{range .Cursor}} {{.}} {{end}}`
var orderBy = `{{if not .Orderby}} {{else}} {{range $idx, $val := .Orderby}} {{.}} {{end}} {{end}}` + limit_tmpl
var select_tmpl = `SELECT {{range $idx, $val := .Columns}} {{if eq $idx 0}} {{else}},{{end}} {{$val}} {{end}} FROM {{.Name}} {{if not .Where}}{{else}} WHERE {{range .Where}} {{.}}{{end}} {{end}}` + orderBy

const (
	subquery_IN  = "subquery_IN"
	subquery_COL = "subquery_COL"
	root_Q       = "ROOT_QUERY"
)

var specialAttributes []string = []string{
	"orderby",
	"limit",
	"offset",
}

var sqlFuncAttributes []string = []string{
	"count",
	"max",
	"avg",
	"sum",
}

// const specialAttributes []string = []string{}

type SelectQuery struct {
	tbl *SelectTbl
}

type SelectTbl struct {
	Name       string
	Alias      string
	Kind       string
	Parent     string
	Columns    []string
	Where      []string
	Wherekeys  []string
	Orderby    []string
	children   []*SelectTbl
	Cursor     []string
	CursorKeys []string
	TmplStr    string
	InnerTmpl  string
}

func createSelectTbl(_json string) *SelectQuery {
	squery := &SelectQuery{}
	gjson.Get(_json, "@this").ForEach(func(tblname, _tblJson gjson.Result) bool {
		tbl := &SelectTbl{
			Kind: root_Q,
			Name: tblname.Str,
		}
		tbl.Parse(_tblJson.Raw)
		squery.tbl = tbl
		return true
	})

	if len(squery.tbl.children) > 0 {
		children := squery.tbl.children
		for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
			children[i], children[j] = children[j], children[i]
		}
	}
	squery.tbl.TmplStr = TmplExecute(select_tmpl, squery.tbl)
	squery.tbl.Wherekeys = append(squery.tbl.Wherekeys, squery.tbl.CursorKeys...)
	return squery
}

func (tbl *SelectTbl) Parse(_json string) {
	gjson.Get(_json, "@this").ForEach(func(_column, _columnJson gjson.Result) bool {
		if _columnJson.Type == gjson.JSON {
			if strings.Contains(_column.Str, "^") {
				name := strings.Split(replaceAllDirty(_column.Str), "_")[0]
				innertbl := &SelectTbl{
					Name:  name,
					Alias: replaceAllDirty(_column.Str),
					Kind:  getTblKind(_column.Str),
				}
				innertbl.Parse(_columnJson.Raw)
				innertbl.TmplStr = TmplExecute(select_tmpl, innertbl)
				innertbl.InnerTmpl = fmt.Sprintf("(%s) %s", innertbl.TmplStr, innertbl.Alias)
				tbl.Columns = append(tbl.Columns, innertbl.InnerTmpl)
				tbl.Wherekeys = append(innertbl.Wherekeys, tbl.Wherekeys...)
				tbl.children = append(tbl.children, innertbl)
				return true
			}
			if strings.Contains(_column.Str, "#") {
				name := strings.Split(replaceAllDirty(_column.Str), "_")[0]
				tblname := strings.Split(replaceAllDirty(_column.Str), "_")[1]
				innertbl := &SelectTbl{
					Name:  tblname,
					Alias: replaceAllDirty(_column.Str),
					Kind:  getTblKind(_column.Str),
				}
				innertbl.Parse(_columnJson.Raw)
				innertbl.TmplStr = TmplExecute(select_tmpl, innertbl)
				innertbl.InnerTmpl = fmt.Sprintf("%s %s IN (%s)", getSeparator(_column.Str), name, innertbl.TmplStr)
				tbl.children = append(tbl.children, innertbl)
				tbl.Where = append(tbl.Where, innertbl.InnerTmpl)
				tbl.Wherekeys = append(tbl.Wherekeys, innertbl.Wherekeys...)
				return true
			}
		}
		if contains(specialAttributes, _column.Str) {
			if _column.Str == "orderby" {
				tbl.Orderby = append(tbl.Orderby, "ORDER BY ?")
				tbl.Wherekeys = append(tbl.Wherekeys, replaceAllDirty(_column.Str))
				return true
			}

			if _column.Str == "limit" || _column.Str == "offset" {
				toUpper := strings.ToUpper(_column.Str)
				toUpper += " ?"
				tbl.Cursor = append(tbl.Cursor, toUpper)
				tbl.CursorKeys = append(tbl.CursorKeys, replaceAllDirty(_columnJson.Str))
				return true
			}
		}
		if isOuterQueryRef(_columnJson.Str) {
			tbl.Where = append(tbl.Where, getSeparator(_columnJson.Str)+" "+_column.Str+" "+getOperator(_columnJson.Str)+" "+replaceAllDirty(_columnJson.Str))
			return true
		}

		if isVariable(_columnJson.Str) {
			tbl.Where = append(tbl.Where, getSeparator(_columnJson.Str)+" "+tbl.Name+"."+_column.Str+" "+getOperator(_columnJson.Str)+" "+"?")
			tbl.Wherekeys = append(tbl.Wherekeys, replaceAllDirty(_columnJson.Str))
			if !strings.Contains(_columnJson.Str, "..") {
				tbl.Columns = append(tbl.Columns, replaceAllDirty(tbl.Name+"."+_column.Str))
			}
			return true
		}

		if contains(sqlFuncAttributes, _column.Str) {
			column := fmt.Sprintf("%s(%s) as %s_%s", _column.Str, _columnJson.Str, _column.Str, _columnJson.Str)
			tbl.Columns = append(tbl.Columns, column)
			return true
		}

		if !strings.Contains(_columnJson.Str, "..") {
			tbl.Columns = append(tbl.Columns, fmt.Sprintf("%s.%s", tbl.Name, _column))
		}
		if strings.Contains(_columnJson.Str, "@") {
			tbl.Where = append(tbl.Where, getSeparator(_columnJson.Str)+_column.Str+" "+getOperator(_columnJson.Str)+" "+"?")
			tbl.Wherekeys = append(tbl.Wherekeys, replaceAllDirty(_columnJson.Str))
		}
		return true
	})
}

func getTblKind(key string) string {
	if strings.Contains(key, "^") {
		return subquery_COL
	}

	if strings.Contains(key, "#") {
		return subquery_IN
	}

	return ""
}

func isOuterQueryRef(value string) bool {
	return strings.Contains(value, ":")
}

func isVariable(value string) bool {
	return strings.Contains(value, "@")
}

func (squery *SelectQuery) Execute(db *sql.DB, data map[string]interface{}) map[string][]map[string]interface{} {
	flat := map[string][]map[string]interface{}{}
	tbl := squery.tbl
	stmt, err := db.Prepare(tbl.TmplStr)
	handleErr(err)

	input := mapData(tbl.Wherekeys, data)
	rows, err := stmt.Query(input...)
	if err != nil {
		log.Printf("here %s", err)
	}

	cols, _ := rows.Columns()
	defer rows.Close()
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			if m := handleErr(err); m != nil {
				flat["err"] = append(flat["err"], m)
				return flat
			}
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}

		flat[tbl.Name] = append(flat[tbl.Name], m)
	}
	handleErr(err)
	return flat
}

func mapData(dataVars []string, data map[string]interface{}) []interface{} {
	ret := make([]interface{}, len(dataVars))
	for idx, vars := range dataVars {
		val, ok := data[vars]
		if ok {
			ret[idx] = val
		}
	}

	return ret
}
