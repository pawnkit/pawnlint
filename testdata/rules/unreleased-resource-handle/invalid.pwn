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

stock LoadGuardedLeak()
{
	new File:file;
	file = fopen("leaky.log");
	if (file)
	{
		fread(file, buffer, 32);
	}
}
