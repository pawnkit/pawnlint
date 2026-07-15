format(output[], size, const pattern[], value)
{
    output[0] = value;
    return size + pattern[0];
}

Check(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", value);
    }
    Consume(output);
}
