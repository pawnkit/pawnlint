new File:global_file = fopen("server.log");

stock File:CustomOpen(const name[])
{
	return File:0;
}

main()
{
	new File:custom = CustomOpen("server.log");
	static File:cached = fopen("cache.log");
	while (true)
	{
		new File:resident = fopen("resident.log");
	}
}
