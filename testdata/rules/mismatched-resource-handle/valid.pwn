main()
{
	fclose(fopen("server.log"));
	DB_Close(DB_Open("server.db"));
	DB_FreeResultSet(DB_ExecuteQuery(DB:1, "SELECT 1"));

	new File:file = ftemp();
	fclose(file);
}
