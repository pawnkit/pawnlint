forward OnDone(playerid, Float:amount);
public OnDone(playerid, Float:amount)
{
    return playerid + floatround(amount);
}

main()
{
    SetTimerEx("OnDone", 1000, false, "df", 0, 1.5);
    SetTimerEx("OnDone", 1000, false, "i", 0);
    SetTimerEx("OnDone", 1000, false, "s", "hello");
    SetTimerEx("OnDone", 1000, false, "b", true);

    new fmt[8] = "i";
    SetTimerEx("OnDone", 1000, false, fmt, 0);
}
