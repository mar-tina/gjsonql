package gjsonql

import (
	"bytes"
	"html/template"
	"log"
	"strings"

	"github.com/lithammer/shortuuid"
)

func newTmpl(id, renderStr string) *template.Template {
	tmpl, err := template.New(id).Parse(renderStr)
	if err != nil {
		log.Printf("could not create template: %s", err)
	}

	return tmpl
}

func TmplExecute(tmplPath string, data interface{}) string {
	id := shortuuid.New()
	tmpl := newTmpl(id, tmplPath)
	out := &bytes.Buffer{}
	tmpl.Execute(out, data)
	return out.String()
}

func getOperator(path string) string {
	if strings.Contains(path, "=") {
		return "="
	}

	if strings.Contains(path, "*") {
		return "LIKE"
	}

	return ""
}

func getSeparator(path string) string {
	if strings.Contains(path, "&") {
		return "AND"
	}

	if strings.Contains(path, "?") {
		return "OR"
	}

	return ""
}

func replaceAllDirty(path string) string {
	minusOR := strings.ReplaceAll(path, "?", "")
	minusAnd := strings.ReplaceAll(minusOR, "&", "")
	minusEq := strings.ReplaceAll(minusAnd, "=", "")
	minusAt := strings.ReplaceAll(minusEq, "@", "")
	minusLike := strings.ReplaceAll(minusAt, "%", "")
	minusAndStr := strings.ReplaceAll(minusLike, "AND", "")
	minusHash := strings.ReplaceAll(minusAndStr, "#", "")
	minusExcl := strings.ReplaceAll(minusHash, "#", "")
	minusColon := strings.ReplaceAll(minusExcl, ":", "")
	minusUp := strings.ReplaceAll(minusColon, "^", "")
	minusStar := strings.ReplaceAll(minusUp, "*", "")
	minusDots := strings.ReplaceAll(minusStar, "..", "")
	return minusDots
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func handleErr(err error) map[string]interface{} {
	if err != nil {
		log.Printf("exec failed: %s", err)
		return map[string]interface{}{
			"err": err,
		}
	}

	return nil
}
