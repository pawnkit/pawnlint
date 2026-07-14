main()
{
    if (IsPlayerConnected(playerid))
    {
        Kick(playerid);
    }

    while (running)
    {
        DoStuff();
    }

    for (new i = 0; i < 10; i++)
    {
        Sum += i;
    }
}