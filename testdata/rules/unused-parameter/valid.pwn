public OnPlayerConnect(playerid)
{
    return 1;
}

stock Add(left, right)
{
    return left + right;
}

stock Ignore(_value)
{
    return 1;
}

CMD:ExternalCommand(playerid, params[])
{
    return 1;
}

hook OnPlayerEnterVehicle(playerid, vehicleid, ispassenger)
{
    return playerid;
}

inline Response(pid, dialogid, response, listitem, string:inputtext[])
{
    return pid;
}

stock PragmaUnused(a, b, c)
{
    #pragma unused b, c
    return a;
}
