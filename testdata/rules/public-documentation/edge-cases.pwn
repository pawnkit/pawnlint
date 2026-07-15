// Ordinary comments are not API documentation.
public OnOrdinaryComment()
{
    return 1;
}

/** Detached documentation. */

public OnDetached()
{
    return 1;
}

/**
 * Uses directed parameters.
 * @param[in] value Input value.
 * @return The result.
 */
stock API_DirectedParameter(value)
{
    return value;
}

/**
 * Handles variable arguments.
 * @param format Format string.
 * @return The result.
 */
stock API_Variadic(const format[], ...)
{
    return format[0];
}
