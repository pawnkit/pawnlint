#define PURE_CALL(%0) PureNative(%0)
native PureNative(value);

Check(limit)
{
    new total;
#if UNKNOWN_FEATURE
    while (limit > 0) {
        total += PureNative(4);
    }
#endif
    while (limit > 0) {
        total += PURE_CALL(4);
        limit--;
    }
    return total;
}
