Update(playerid)
{
    new elapsed = GetTickCountDifference(GetTickCount(), lastAction[playerid]);
    return elapsed;
}

new lastAction[500];

StorePastTimestamp()
{
    new due = GetTickCount() + 1000;
    return due;
}

forward GetTickCountDifference(a, b);
