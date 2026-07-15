Check()
{
    new value = 1;
#if UNKNOWN_FEATURE
    while (value)
        print("unknown");
#endif
    while (value)
    {
        value = 0;
    }
}
