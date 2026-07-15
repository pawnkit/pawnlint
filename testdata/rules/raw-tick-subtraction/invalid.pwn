CheckCooldown(playerid)
{
    if (GetTickCount() - lastAction[playerid] > 1000)
    {
        return 1;
    }
    return 0;
}

new lastAction[500];

StoreDuration(started)
{
    new duration = GetTickCount() - started;
    return duration;
}
