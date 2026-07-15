native GoodNative();
native ReservedFunction();

new const RELEASE_BUILD = 1;

enum Status
{
	STATUS_OK
}

public ProcessData()
{
	return 1;
}

stock CalculateSpeed(Float:speed, temporaryAllowed)
{
	new result = _:speed;
	return result + temporaryAllowed;
}

main()
{
	CalculateSpeed(Float:1, 0);
}
