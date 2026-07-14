main()
{
	Direct(2);
	First();
}

stock Direct(value)
{
	if (value > 0)
	{
		return Direct(value - 1);
	}
	return 0;
}

stock First()
{
	Second();
}

stock Second()
{
	First();
}
