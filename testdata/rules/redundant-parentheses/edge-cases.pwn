main()
{
    new value = (/* keep */ 1);
    if (((value = 1)))
    {
        value++;
    }
    return value;
}

#if UNKNOWN_FEATURE
main()
{
    return (1);
}
#endif
