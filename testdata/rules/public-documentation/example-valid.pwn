/**
 * Creates an account.
 * @param playerid Player identifier.
 * @param name Account name.
 * @return Non-zero on success.
 */
stock API_CreateAccount(playerid, const name[])
{
    return playerid + name[0];
}
