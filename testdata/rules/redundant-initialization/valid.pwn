native Use(value);
native SideEffect();

main()
{
	ReadBeforeWrite();
	PartialWrite(true);
	Effectful();
	ReadWrite();
	NoOverwrite();
	LoopDeclaration();
	StaticLocal();
	SameDeclaration();
}

stock ReadBeforeWrite()
{
	new value = 0;
	Use(value);
	value = 1;
}

stock PartialWrite(bool:condition)
{
	new value = 0;
	if (condition)
	{
		value = 1;
	}
	Use(value);
}

stock Effectful()
{
	new value = SideEffect();
	value = 1;
	Use(value);
}

stock ReadWrite()
{
	new value = 0;
	value += 1;
	Use(value);
}

stock NoOverwrite()
{
	new value = 0;
}

stock LoopDeclaration()
{
	for (new index = 0; index < 2; index++)
	{
		Use(index);
	}
}

stock StaticLocal()
{
	static value = 0;
	value = 1;
}

stock SameDeclaration()
{
	new value = 0, copy = value;
	value = 1;
	Use(copy);
}
