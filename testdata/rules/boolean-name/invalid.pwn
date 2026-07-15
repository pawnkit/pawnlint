new bool:enabled;

bool:Ready()
{
	return true;
}

stock UpdatePlayer(bool:active)
{
	new bool:visible = active;
	new bool:island = true;
	return visible && island;
}

main()
{
	UpdatePlayer(Ready());
}
