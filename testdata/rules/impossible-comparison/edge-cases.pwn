Check(value)
{
#if UNKNOWN_FEATURE
    if ((value & 3) > 7)
        return 1;
#endif
    if (value % cellmin > 10)
        return 1;
    if (-(value & 0xFFFFFFFF) > 1)
        return 1;
    return 0;
}
