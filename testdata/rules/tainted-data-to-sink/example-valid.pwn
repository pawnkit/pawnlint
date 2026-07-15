native SQL_Query(const query[]);

public OnPluginInput(playerid, const text[])
{
    SQL_Query("SELECT id FROM players");
    return playerid + text[0];
}
