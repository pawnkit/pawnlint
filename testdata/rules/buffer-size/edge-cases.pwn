GetPlayerName(playerid, output[], size) {}

main()
{
    new name[8];
	new address[8];
    new dynamic_size = 32;
    GetPlayerName(0, name, dynamic_size);
	GetPlayerIp(0, address, dynamic_size);
}
