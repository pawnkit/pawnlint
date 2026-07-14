main(bool:condition)
{
	new File:file = fopen("first.log");
	fclose(file);
	file = fopen("second.log");

	new DB:database = DB_Open("first.db");
	ConsumeDatabase(database);
	database = DB_Open("second.db");

	new DBResult:result = DB_ExecuteQuery(database, "SELECT 1");
	if (condition)
	{
		UseCondition();
	}
	result = DB_ExecuteQuery(database, "SELECT 2");
}

stock ReplaceClosedFile(bool:condition)
{
	new File:file = fopen("first.log");
	if (condition)
	{
		fclose(file);
	}
	else
	{
		fclose(file);
	}
	file = ftemp();
}
