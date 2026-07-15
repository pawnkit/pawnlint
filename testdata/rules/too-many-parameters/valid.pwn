Allowed(first, second, third)
{
    return first + second + third;
}

public OnExternalCallback(first, second, third, fourth, fifth)
{
    return first + second + third + fourth + fifth;
}

OnPlayerWeaponShot(playerid, weaponid, hittype, hitid, Float:x, Float:y, Float:z)
{
    return playerid + weaponid + hittype + hitid + floatround(x + y + z);
}

Generated_Interface(first, second, third, fourth, fifth)
{
    return first + second + third + fourth + fifth;
}

Variadic(const format[], ...)
{
    return format[0];
}
