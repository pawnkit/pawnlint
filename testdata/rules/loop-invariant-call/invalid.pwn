native PureNative(value);

PureFunction(value)
{
    return value + 1;
}

Check(limit)
{
    new total;
    for (new i; i < limit; i++) {
        total += PureNative(limit);
        total += PureFunction(4);
    }
    return total;
}
