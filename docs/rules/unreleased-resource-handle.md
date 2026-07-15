# unreleased-resource-handle

Reports local resource handles that can reach function exit without release

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | control-flow |
| Default | disabled |
| Fixable | no |
| Tags | resource, handle, database, file, control-flow |

## Details

A local initialized from a known resource creator must be released on every path. The rule follows definite scalar aliases and simple project wrappers, then stops when ownership becomes ambiguous.

## Configuration

```toml
[rules]
unreleased-resource-handle = "warning"
```

## Examples

### Bad

```pawn
main()
{
	new File:file = fopen("server.log");
	new File:temporary = ftemp();
	new DB:database = DB_Open("server.db");
	new DBResult:result = DB_ExecuteQuery(DB:1, "SELECT 1");
}

stock WriteFile(bool:condition)
{
	new File:file = fopen("output.log");
	fwrite(file, "entry");
	if (condition)
	{
		fclose(file);
	}
}

stock AssignFile()
{
	new File:file;
	file = fopen("assigned.log");
}
// …
```

### Good

```pawn
stock File:OpenLog()
{
	new File:file = fopen("server.log");
	return file;
}

main()
{
	new DB:database = DB_Open("server.db");
	DB_Close(database);

	new DBResult:result = DB_ExecuteQuery(DB:1, "SELECT 1");
	ConsumeResult(result);
}

stock UseFile(bool:condition)
{
	new File:file = fopen("server.log");
	if (condition)
	{
		fclose(file);
	}
	else
	{
		fclose(file);
	}
}
// …
```
