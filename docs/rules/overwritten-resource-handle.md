# overwritten-resource-handle

Reports resource handles overwritten before any use or release

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file, control-flow |

## Details

Replacing a local file or SQLite handle loses the previous resource. The rule reports only two direct acquisitions connected by one linear control-flow path with no intervening reference.

## Configuration

```toml
[rules]
overwritten-resource-handle = "warning"
```

## Examples

### Bad

```pawn
main()
{
	new File:file = fopen("first.log");
	new unrelated = 1;
	file = ftemp();

	new DB:database;
	database = DB_Open("first.db");
	database = db_open("second.db");

	new DBResult:result = db_query(database, "SELECT 1");
	result = DB_ExecuteQuery(database, "SELECT 2");
}

stock ReplaceFile(bool:condition)
{
	new File:file = fopen("first.log");
	fwrite(file, "entry");
	if (condition)
	{
		file = ftemp();
	}
}
```

### Good

```pawn
main()
{
    new File:log = fopen("first.log");
    fclose(log);
    log = fopen("second.log");
    fclose(log);
}
```
