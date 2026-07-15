forward OnRespawn(playerid, hospitalid);

main(playerid)
{
    SetTimerEx("OnRespawn", 1000, false, "ii", playerid);
}
