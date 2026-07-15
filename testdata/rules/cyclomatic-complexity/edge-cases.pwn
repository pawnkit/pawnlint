ConditionalBuild(value)
{
    #if defined UNKNOWN_FEATURE
        if (value && value > 1)
        {
            return 1;
        }
    #endif
    return value;
}

InactiveBuild(value)
{
    #if 0
        if (value || value > 1)
        {
            return 1;
        }
    #endif
    return value;
}

NestedSwitch(value)
{
    switch (value)
    {
        case 1, 2: return 1;
        default: return 0;
    }
}
