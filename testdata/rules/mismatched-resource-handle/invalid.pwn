main()
{
	fclose(DB_Open("server.db"));
	DB_Close(fopen("server.log"));
	DB_FreeResultSet(DB_Open("server.db"));

	new DB:database = DB_Open("server.db");
	fclose(database);
}
