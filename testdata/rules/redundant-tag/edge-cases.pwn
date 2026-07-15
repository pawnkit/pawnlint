UseMixed({Float,_}:value)
{
    return Float:value;
}

main()
{
    new Float:value = 1.0;
    return Float:/* preserve */value;
}

#if UNKNOWN_FEATURE
main()
{
    new Float:value = 1.0;
    return Float:value;
}
#endif
