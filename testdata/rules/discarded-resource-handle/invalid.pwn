main()
{
	fopen("server.log");
	ftemp();
	db_open("server.db");
	db_query(DB:1, "SELECT 1");
	DB_Open("server.db");
	DB_ExecuteQuery(DB:1, "SELECT 1");
}
