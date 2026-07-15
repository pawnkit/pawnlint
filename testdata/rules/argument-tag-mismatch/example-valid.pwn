TeleportPlayer(playerid, Float:x, Float:y, Float:z)
{
    SetPlayerPos(playerid, x, y, z);
}

HandleTeleport(playerid, targetid)
{
    new Float:x, Float:y, Float:z;
    GetPlayerPos(targetid, x, y, z);
    TeleportPlayer(playerid, x, y, z);
}
