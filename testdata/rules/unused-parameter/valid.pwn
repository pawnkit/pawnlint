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
