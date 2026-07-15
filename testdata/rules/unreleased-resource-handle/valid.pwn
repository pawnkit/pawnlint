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

stock AssignAndCloseFile()
{
	new File:file;
	file = fopen("assigned.log");
	fclose(file);
}

stock LoadGuarded()
{
	new File:file;
	file = fopen("guarded.log");
	if (file)
	{
		fclose(file);
	}
}

stock LoadEarlyReturn()
{
	new File:file;
	file = fopen("guarded2.log");
	if (!file)
	{
		return;
	}
	fclose(file);
}
