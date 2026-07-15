ProcessPlayer(playerid)
{
    if (!IsPlayerConnected(playerid))
    {
        return 0;
    }
    if (GetPlayerState(playerid) == PLAYER_STATE_WASTED)
    {
        Kick(playerid);
    }
    return 1;
}
