CheckChanging(limit)
{
    new output[64];
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", i);
    }
    Consume(output);
}

CheckConsumed(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", value);
        Consume(output);
    }
}

CheckConditional(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        if (i > 2) {
            format(output, sizeof output, "value %d", value);
        }
    }
    Consume(output);
}

CheckControlled(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        if (i == 2) continue;
        format(output, sizeof output, "value %d", value);
    }
    Consume(output);
}

CheckUnused(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", value);
    }
}

Format(output[], size, const pattern[], value)
{
    output[0] = value;
    return size + pattern[0];
}

CheckShadowed(limit, value)
{
    new output[64];
    for (new i; i < limit; i++) {
        Format(output, sizeof output, "value %d", value);
    }
    Consume(output);
}
