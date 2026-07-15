ConditionalExit()
{
    while (1)
    {
#if UNKNOWN_FEATURE
        break;
#endif
        print("unknown");
    }
}

UnreachableBreak()
{
    while (1)
    {
        if (0)
            break;
        print("forever");
    }
}
