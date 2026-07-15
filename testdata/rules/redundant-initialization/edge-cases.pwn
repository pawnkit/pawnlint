native Use(value);

main()
{
	ArrayValue();
	ConditionalWrite(true);
}

stock ArrayValue()
{
	new values[2] = {0, 1};
	values[0] = 2;
	Use(values[0]);
}

stock ConditionalWrite(bool:condition)
{
	new value = 0;
	condition && (value = 1);
	Use(value);
}
