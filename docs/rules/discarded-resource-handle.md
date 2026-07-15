# discarded-resource-handle

Reports resource handles discarded before they can be released

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file |

## Details

File, database, and database-result creators return handles that must be closed or freed. The rule reports direct standalone calls whose returned handle is immediately lost.

## Configuration

```toml
[rules]
discarded-resource-handle = "warning"
```

## Examples

### Bad

```pawn
main()
{
	fopen("server.log");
	ftemp();
	db_open("server.db");
	db_query(DB:1, "SELECT 1");
	DB_Open("server.db");
	DB_ExecuteQuery(DB:1, "SELECT 1");
}
```

### Good

```pawn
main()
{
	new File:file = fopen("server.log");
	fclose(file);

	new DB:database = DB_Open("server.db");
	new DBResult:result = DB_ExecuteQuery(database, "SELECT 1");
	DB_FreeResultSet(result);
	DB_Close(database);
}
```
