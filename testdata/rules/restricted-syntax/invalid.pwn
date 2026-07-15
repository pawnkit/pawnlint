#include "restricted.inc"

native RestrictedNative();

new forbiddenGlobal;

stock LegacyFunction()
{
	return 1;
}

stock Recur()
{
	Recur();
	return 1;
}

main()
{
	LegacyFunction();
	RestrictedNative();
	goto finished;
finished:
	return forbiddenGlobal;
}
