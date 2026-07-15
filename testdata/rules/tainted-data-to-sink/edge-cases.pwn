#define FORWARD_INPUT(%0) SQL_Query(%0)

public OnPluginInput(playerid, const text[])
{
    FORWARD_INPUT(text);
#if UNKNOWN_FEATURE
    SQL_Query(text);
#endif
    return playerid;
}
