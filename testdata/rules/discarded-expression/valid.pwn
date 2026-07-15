enum E_VALUES
{
    E_VALUE
}

main()
{
    Kick(playerid);
    x = 10;
    y++;
    z--;
    ++y;
    --z;
    foo(a, b, c);
    bar(x);
    return callcmd::goto(playerid, params);
}

StopTimers()
{
    stop timerid;
    stop timers[playerid];
}
