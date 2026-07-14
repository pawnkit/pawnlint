# unescaped-sql-format

Reports mysql_format calls using %s for a non-literal string argument

| | |
| --- | --- |
| Category | correctness |
| Severity | warning |
| Analysis | semantic |
| Default | disabled |
| Fixable | no |
| Tags | sql, security, format, native |

## Details

mysql_format inserts `%s` arguments into the query exactly as given, while
`%e` escapes them first. A non-literal `%s` argument that carries a player
name, chat input, or any other untrusted value is a SQL injection risk:

```pawn
mysql_format(handle, query, sizeof(query), "SELECT * FROM users WHERE name = '%s'", name);
```

Use `%e` instead so the value is escaped:

```pawn
mysql_format(handle, query, sizeof(query), "SELECT * FROM users WHERE name = '%e'", name);
```

The rule only flags `%s` arguments that are not a literal string written
directly in the call, since a literal cannot carry untrusted input. It only
recognizes calls literally named `mysql_format`. No fix is offered because
some `%s` arguments are genuinely safe (e.g. a hardcoded column name).
