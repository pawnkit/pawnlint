stock File:CustomOpen(const name[])
{
	return File:0;
}

main()
{
	static File:cached = fopen("first.log");
	cached = ftemp();

	new File:custom = CustomOpen("first.log");
	custom = fopen("second.log");
}
