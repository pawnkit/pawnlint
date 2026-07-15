ProcessPlayers(limit)
{
    for (new playerid; playerid < limit; playerid++)
    {
        if (IsPlayerConnected(playerid))
        {
            while (GetPlayerState(playerid) == PLAYER_STATE_WASTED)
            {
                Kick(playerid);
            }
        }
    }
}
