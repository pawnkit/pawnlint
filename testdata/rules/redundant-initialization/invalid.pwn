native Use(value);

main()
{
	Immediate();
	BothBranches(true);
	EarlyExit(true);
	DoLoop(true);
}

stock Immediate()
{
	new value = 0;
	value = 1;
	Use(value);
}

stock BothBranches(bool:condition)
{
	new value = 10;
	if (condition)
	{
		value = 1;
	}
	else
	{
		value = 2;
	}
	Use(value);
}

stock EarlyExit(bool:condition)
{
	new value = 20;
	if (condition)
	{
		return;
	}
	value = 2;
	Use(value);
}

stock DoLoop(bool:condition)
{
	new value = 30;
	do
	{
		value = 3;
	}
	while (condition);
	Use(value);
}
