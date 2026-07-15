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

/// Handles a connection.
/// @param playerid Player identifier.
/// @return Non-zero on success.
public OnPlayerConnect(playerid)
{
    return 1;
}

public OnInternal()
{
    return 1;
}

stock Helper()
{
    return 1;
}
