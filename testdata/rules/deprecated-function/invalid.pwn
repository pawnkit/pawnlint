main()
{
	new players = GetPlayerPoolSize();
	new vehicles = GetVehiclePoolSize();
	new actors = GetActorPoolSize();
	return players + vehicles + actors;
}
