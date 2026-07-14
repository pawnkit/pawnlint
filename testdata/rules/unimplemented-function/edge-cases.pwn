#define EnableTirePopping(%0) CustomTireHandling(%0)

stock SetDeathDropAmount(amount)
{
	return amount;
}

native SetTeamCount(count);

main()
{
	EnableTirePopping(false);
	SetDeathDropAmount(100);
	SetTeamCount(4);
}
