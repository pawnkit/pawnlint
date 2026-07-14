#define fopen(%0) CustomOpen(%0)

stock DB:DB_Open(const name[])
{
	return DB:0;
}

main()
{
	fopen("server.log");
	DB_Open("server.db");
	DB_FreeResultSet(DB_ExecuteQuery(DB:1, "SELECT 1"));
}
