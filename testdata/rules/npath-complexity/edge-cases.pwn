ConditionalBuild(value)
{
    #if defined UNKNOWN_FEATURE
        if (value)
        {
            if (value > 1)
            {
                return 1;
            }
        }
    #endif
    return value;
}

InactiveBuild(value)
{
    #if 0
        if (value)
        {
            if (value > 1)
            {
                return 1;
            }
        }
    #endif
    return value;
}

SwitchWithoutDefault(value)
{
    switch (value)
    {
        case 1: return 1;
        case 2: return 2;
    }
    return 0;
}
