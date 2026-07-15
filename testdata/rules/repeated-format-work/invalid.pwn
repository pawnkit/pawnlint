CheckFormat(limit, value)
{
    new output[64];
    new total;
    for (new i; i < limit; i++) {
        format(output, sizeof output, "value %d", value);
        total += i;
    }
    Consume(output);
    return total;
}

CheckStrformat(limit)
{
    new output[64];
    while (limit-- > 0) {
        strformat(output, sizeof output, false, "ready");
    }
    Consume(output);
}

CheckOpenMPFormat(limit)
{
    new output[64];
    do {
        Format(output, sizeof output, "ready");
    } while (--limit > 0);
    Consume(output);
}
