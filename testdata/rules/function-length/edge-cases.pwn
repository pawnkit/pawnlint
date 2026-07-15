ConditionalBuild(value)
{
    #if defined UNKNOWN_FEATURE
        if (value)
        {
            value--;
            value--;
            value--;
            value--;
        }
    #endif
    return value;
}

InactiveBuild(value)
{
    #if 0
        if (value)
        {
            value--;
            value--;
            value--;
            value--;
        }
    #endif
    return value;
}
