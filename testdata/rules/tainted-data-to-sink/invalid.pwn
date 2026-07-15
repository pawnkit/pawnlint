public OnPluginInput(playerid, const text[])
{
    new query[128];
    format(query, sizeof query, "SELECT '%s'", text);
    SQL_Query(query);
    ForwardInput(text);
    new command[64];
    Plugin_Read(command, sizeof command);
    ExecuteCommand(command);
    return playerid;
}

ForwardInput(const value[])
{
    OpenPath(value);
}
