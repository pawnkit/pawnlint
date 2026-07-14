stock SetPlayerAdmin(playerid, bool:admin)
{
	return playerid + admin;
}

native ClearBanList();

main()
{
	SetPlayerAdmin(0, true);
	ClearBanList();
}
