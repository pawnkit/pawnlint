#define GetPlayerPoolSize() CustomPlayerCount()

stock GetVehiclePoolSize()
{
	return 0;
}

native GetActorPoolSize();

main()
{
	new players = GetPlayerPoolSize();
	new vehicles = GetVehiclePoolSize();
	new actors = GetActorPoolSize();
	return players + vehicles + actors;
}
