package gjsonql

func Parse(action, queryName, data string) {
	switch action {
	case SELECT:
		createSelectTbl(data)
	}
}
