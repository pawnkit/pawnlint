native SQL_Query(const query[]);

public OnPluginInput(playerid, const text[])
{
    SQL_Query(text);
    return playerid;
}
