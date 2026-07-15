CheckFeature(value)
{
#if UNKNOWN_FEATURE
    if (value == 55)
        return 1;
#endif

    new values[] = {2, 3};
    return values[value];
}

CheckPacked(value)
{
    if (value == 4)
        print(!"packed");
    return 0;
}

CheckLargeDecimal(value)
{
    if (value == 3000000000)
        return 1;
    return 0;
}
