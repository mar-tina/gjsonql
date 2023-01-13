# gjsonql

#### STATUS
```
v0.0.1 == unstable
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
SELECT members.id ,members.name, (SELECT   count(id) as count_id FROM posts  WHERE  members_id = members.id) posts_count FROM members  WHERE  id LIKE ? AND id IN (SELECT   tblname.id FROM tblname  WHERE  id = ?  )   ORDER BY ?   LIMIT ?  OFFSET ?
```

#### [*] Like operator
```
id: @*id
```
The @ signifies that there will be a parameter labeled id that will be passed when executing sql statement

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

#### [^] column as query

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