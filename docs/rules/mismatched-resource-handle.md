# mismatched-resource-handle

Reports handles passed to the wrong resource releaser

| | |
| --- | --- |
| Category | correctness |
| Severity | error |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file, tag |

## Details

File, database, and database-result handles have distinct release functions. The rule reports calls only when the argument has one definite incompatible resource tag.

## Configuration

```toml
[rules]
mismatched-resource-handle = "error"
```

## Examples

### Bad

```pawn
main()
{
	fclose(DB_Open("server.db"));
	DB_Close(fopen("server.log"));
	DB_FreeResultSet(DB_Open("server.db"));

	new DB:database = DB_Open("server.db");
	fclose(database);
}
```

### Good

```pawn
main()
{
	fclose(fopen("server.log"));
	DB_Close(DB_Open("server.db"));
	DB_FreeResultSet(DB_ExecuteQuery(DB:1, "SELECT 1"));

	new File:file = ftemp();
	fclose(file);
}
```
