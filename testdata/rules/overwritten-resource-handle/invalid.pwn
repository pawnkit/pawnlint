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
