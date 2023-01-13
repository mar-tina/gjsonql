package gjsonql

import "database/sql"

type Query interface {
	Execute(db *sql.DB, data map[string]interface{}) map[string][]map[string]interface{}
}

var inmemStore map[string]Query = map[string]Query{}

func Parse(action, queryName, data string) Query {
	switch action {
	case SELECT:
		query := createSelectTbl(data)
		inmemStore[queryName] = query
		return query
	case INSERT:
		query := Insert(data)
		inmemStore[queryName] = query
		return query
	case CREATE:
		query := Create(data)
		inmemStore[queryName] = query
		return query
	case UPDATE:
		query := Update(data)
		inmemStore[queryName] = query
		return query
	default:
		return nil
	}
}

func ExecuteQuery(db *sql.DB, queryName string, data map[string]interface{}) map[string][]map[string]interface{} {
	if q, ok := inmemStore[queryName]; ok {
		return q.Execute(db, data)
	}

	return nil
}
