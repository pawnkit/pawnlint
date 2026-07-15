#define DECLARE_LOCAL(%0) new %0

forward ReservedFunction2();

public ReservedFunction2()
{
	DECLARE_LOCAL(temp_macro);
	return 1;
}

public ProcessData()
{
	return 1;
}

#if 0
stock ProcessData()
{
	return 1;
}
#endif

main()
{
}
