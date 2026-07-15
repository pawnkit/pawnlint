public OnPluginInput(playerid, const text[])
{
    SQL_Query("SELECT 1");
    new query[128];
    query = text;
    query = "SELECT 1";
    SQL_Query(query);
    new command[64];
    Plugin_Read(command, sizeof command);
    Plugin_Clean(command, sizeof command);
    ExecuteCommand(command);
    new unknown[64];
    unknown = text;
    UnknownSanitizer(unknown);
    SQL_Query(unknown);
    new rewritten[64];
    rewritten = text;
    Rewrite(rewritten);
    SQL_Query(rewritten);
    return playerid;
}

Rewrite(value[])
{
    value[0] = EOS;
}

OnPluginInputShadow(const text[])
{
    SQL_Query(text);
}
