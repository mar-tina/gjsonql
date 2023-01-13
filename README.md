# gjsonql

#### STATUS
```
v0.0.1 == unstable
```

#### CREATE
```go
var CreateQ = `{
    "members": {
        "id": "integer|primary",
        "email": "varchar|not null|unique",
        "name": "varchar|not null"
    },
    "posts": {
		"id": "integer|primary",
        "members_id": "integer",
        "content": "varchar",
        "status": "varchar",
        "date": "varchar"
    }
}`
gjsonql.Parse(gjsonql.CREATE, "createNewTables", CreateQ).Execute(db, nil)
```

#### SELECT
```go
var SELECTQ = `{
    "members": {
        "id": "@*id",
        "name": "",
        "^posts_count": {
            "count": "id",
            "members_id": ":=members.id",
        },
        "&#id_tblname": {
            "id": "@=tblid",
        },
        "orderby": "@orderby",
        "limit": "@limit",
        "offset": "@offset"
    }
}`
gjsonql.Parse(gjsonql.SELECT, "provideAnyName", SELECTQ)
```

```sql
SELECT members.id ,members.name, 
(SELECT   count(id) as count_id FROM posts  WHERE  members_id = members.id) posts_count
FROM members  WHERE  id LIKE ? 
AND id IN (SELECT   tblname.id FROM tblname  WHERE  id = ?)
ORDER BY ? LIMIT ? OFFSET ?
```

#### [*] Like operator
```
id: @*id
```
```sql
select id from tbl where id LIKE ? [@id]
```

#### [:] non parameterized condition
```
members_id: :=members.id
```
the query remains unchanged, it will not be parameterized can be seen below

```sql
select count(id) from tbl where members_id = members.id
```

#### [^] query as column

```
^posts_count
```
the prefix in posts_count is the tbl name and suffix is used to name the column alias

```
posts is the name of an existing table
alias will be posts_count as the column name in query
```

#### [#] where in 
```
#id_tblname
```
prefix in #id_tblname is the column and suffix is the tbl name

#### INSERT
```go
var SetNewMember = `
{
    "members":{
        "name": "@name", 
        "email": "@email",
    }
}
`
gjsonql.Parse(gjsonql.INSERT, "insertIntoMembers", SetNewMember).Execute(db, map[string]interface{}{
    "name":  "jane doe",
    "email": "doe@email.com",
}
```

#### UPDATE
```go
var UpdateTbl = `{
	"members": {
		"#members.id_posts": {
			"id": "id",
			"status": "..@=status",
		}
		"email": "email"
	}	
}`
gjsonql.Parse(gjsonql.UPDATE, "updateMembers", UpdateTbl).Execute(db, map[string]interface{}{
    "email": "new email value",
    "id": ""
})
```

```sql
UPDATE members SET email = ? WHERE members.id IN (SELECT posts.id FROM posts WHERE posts.status = ?);
```

#### [..] ignore field
Double dot syntax indicates field should not be fetched and to be ignored.

