public OnMissing(playerid)
{
    return playerid;
}

/** Short. */
stock API_Short()
{
    return 1;
}

/**
 * Documents parameters.
 * @return A value.
 */
stock API_MissingParameter(value)
{
    return value;
}

/**
 * Documents parameters.
 * @param value
 * @return A value.
 */
stock API_EmptyParameter(value)
{
    return value;
}

/**
 * Documents parameters.
 * @param value First value.
 * @param value Second value.
 * @return A value.
 */
stock API_DuplicateParameter(value)
{
    return value;
}

/**
 * Documents parameters.
 * @param other Another value.
 * @return A value.
 */
stock API_UnknownParameter()
{
    return 1;
}

/**
 * Computes a value.
 */
stock API_MissingReturn()
{
    return 1;
}

/**
 * Computes a value.
 * @return
 */
stock API_EmptyReturn()
{
    return 1;
}
