#define DECLARE_BOOL(%0) new bool:%0

native bool:N();
forward bool:Go();

bool:Go()
{
	DECLARE_BOOL(generated);
	return true;
}

public bool:CB()
{
	return true;
}

stock UnionTag({bool,_}:value)
{
	return _:value;
}

#if 0
new bool:inactive;
#endif

main()
{
}
