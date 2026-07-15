#define CALL_LEGACY() LegacyFunction()

stock LegacyFunction()
{
	return 1;
}

#if defined UNKNOWN_POLICY
new uncertainGlobal;
#endif

#if 0
new inactiveGlobal;
#endif

main()
{
	CALL_LEGACY();
}
