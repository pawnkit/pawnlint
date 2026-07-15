new id;

enum PlayerState
{
	PLAYER_STATE_NAME_TOO_LONG
}

stock Do(id)
{
	new x = id;
	new excessivelyLong;
	for (new q = 0; q < 2;)
	{
		excessivelyLong += q++;
	}
	return x + excessivelyLong;
}

main()
{
	Do(1);
}
