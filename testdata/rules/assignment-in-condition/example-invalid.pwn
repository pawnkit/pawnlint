main(playerid)
{
    if (playerid = GetMaxPlayers())
    {
        return 1;
    }
    return 0;
}
