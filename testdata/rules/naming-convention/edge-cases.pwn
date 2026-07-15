#define DECLARE_LOCAL(%0) new %0

forward badForward();

public badForward()
{
	DECLARE_LOCAL(BadMacro);
	return 1;
}

public bad_callback()
{
	return 1;
}

#if 0
stock badInactive()
{
	return 1;
}
#endif

main()
{
}
