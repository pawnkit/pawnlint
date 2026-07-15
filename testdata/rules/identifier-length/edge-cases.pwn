#define DECLARE_LOCAL(%0) new %0

native N();

forward Go();

stock Go()
{
	DECLARE_LOCAL(x);
	return 1;
}

public CB()
{
	return 1;
}

#if 0
new x;
#endif

main()
{
}
