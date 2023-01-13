package gjsonql

import (
	"fmt"
	"log"
	"strings"

	"github.com/tidwall/gjson"
)

//	`{
//		"member": {}
//	}`
var LIMIT_TMPL = `{{range .Cursor}} {{.}} {{end}}`
var orderBy = `{{if not .Orderby}} {{else}} {{range $idx, $val := .Orderby}} {{.}} {{end}} {{end}}` + LIMIT_TMPL
var SELECT_TMPL = `SELECT {{range $idx, $val := .Columns}} {{if eq $idx 0}} {{else}},{{end}}{{$val}}{{end}} FROM {{.Name}} {{if not .Where}}{{else}} WHERE {{range .Where}} {{.}}{{end}} {{end}}` + orderBy

const (
	SUBQUERY_IN  = "SUBQUERY_IN"
	SUBQUERY_COL = "SUBQUERY_COL"
	ROOTQUERY    = "ROOT_QUERY"
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
	tbls []*SelectTbl
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

func createSelectTbl(_json string) {
	squery := &SelectQuery{}
	gjson.Get(_json, "@this").ForEach(func(tblname, _tblJson gjson.Result) bool {
		tbl := &SelectTbl{
			Kind: ROOTQUERY,
			Name: tblname.Str,
		}
		tbl.Parse(_tblJson.Raw)
		squery.tbls = append(squery.tbls, tbl)
		return true
	})

	for _, tbl := range squery.tbls {
		if len(tbl.children) > 0 {
			children := tbl.children
			for i, j := 0, len(children)-1; i < j; i, j = i+1, j-1 {
				children[i], children[j] = children[j], children[i]
			}
			// for _, childTbl := range children {
			// 	if childTbl.Kind == SUBQUERY_COL {
			// 		out := TmplExecute(SELECT_TMPL, childTbl)
			// 		outFmt := fmt.Sprintf("(%s) %s", out, childTbl.Alias)
			// 		tbl.Columns = append(tbl.Columns, outFmt)
			// 		tbl.Wherekeys = append(childTbl.Wherekeys, tbl.Wherekeys...)
			// 	}
			// }
		}
		out := TmplExecute(SELECT_TMPL, tbl)
		tbl.Wherekeys = append(tbl.Wherekeys, tbl.CursorKeys...)
		log.Printf("TBL::::%v %v", out, tbl.Wherekeys)
	}
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
				innertbl.TmplStr = TmplExecute(SELECT_TMPL, innertbl)
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
				innertbl.TmplStr = TmplExecute(SELECT_TMPL, innertbl)
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
			tbl.Where = append(tbl.Where, getSeparator(_columnJson.Str)+_column.Str+" "+getOperator(_columnJson.Str)+" "+replaceAllDirty(_columnJson.Str))
			return true
		}

		if isVariable(_columnJson.Str) {
			tbl.Where = append(tbl.Where, getSeparator(_columnJson.Str)+_column.Str+" "+getOperator(_columnJson.Str)+" "+"?")
			tbl.Wherekeys = append(tbl.Wherekeys, replaceAllDirty(_columnJson.Str))
			tbl.Columns = append(tbl.Columns, replaceAllDirty(tbl.Name+"."+_column.Str))
			return true
		}

		if contains(sqlFuncAttributes, _column.Str) {
			column := fmt.Sprintf("%s(%s) as %s_%s", _column.Str, _columnJson.Str, _column.Str, _columnJson.Str)
			tbl.Columns = append(tbl.Columns, column)
			return true
		}

		tbl.Columns = append(tbl.Columns, fmt.Sprintf("%s.%s", tbl.Name, _column))
		if strings.Contains(_columnJson.Str, "@") {
			tbl.Where = append(tbl.Where, getSeparator(_columnJson.Str)+_column.Str+" "+getOperator(_columnJson.Str)+" "+"?")
			tbl.Wherekeys = append(tbl.Wherekeys, replaceAllDirty(_columnJson.Str))
		}
		return true
	})
}

func getTblKind(key string) string {
	if strings.Contains(key, "^") {
		return SUBQUERY_COL
	}

	if strings.Contains(key, "#") {
		return SUBQUERY_IN
	}

	return ""
}

func isOuterQueryRef(value string) bool {
	return strings.Contains(value, ":")
}

func isVariable(value string) bool {
	return strings.Contains(value, "@")
}
