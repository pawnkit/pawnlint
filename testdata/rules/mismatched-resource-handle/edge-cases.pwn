stock fclose(handle)
{
	return handle;
}

main()
{
	new unknown;
	fclose(unknown);
	db_close(DB:1);
	db_free_result(DBResult:1);
}
