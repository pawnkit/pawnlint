native PureNative(value);
native MutableNative(value);

Check(limit, const text[])
{
    new total;
    for (new i; i < limit; i++) {
        total += PureNative(i);
        total += MutableNative(limit);
        limit--;
        total += floatabs(limit);
        total += strlen(text);
    }
    while ((limit = MutableNative(limit)) > 0) {
        total += PureNative(limit);
    }
    return total;
}
