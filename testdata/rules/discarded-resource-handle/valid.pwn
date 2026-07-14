main()
{
	new File:file = fopen("server.log");
	fclose(file);

	new DB:database = DB_Open("server.db");
	new DBResult:result = DB_ExecuteQuery(database, "SELECT 1");
	DB_FreeResultSet(result);
	DB_Close(database);
}
