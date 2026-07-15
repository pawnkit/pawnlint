ConditionalBuild(value)
{
    #if defined UNKNOWN_FEATURE
        while (value)
        {
            if (value > 1)
            {
                for (new index = 0; index < value; index++)
                {
                    value -= index;
                }
            }
        }
    #endif
    return value;
}

InactiveBuild(value)
{
    #if 0
        while (value)
        {
            if (value > 1)
            {
                for (new index = 0; index < value; index++)
                {
                    value -= index;
                }
            }
        }
    #endif
    return value;
}

Ternary(value)
{
    return value ? (value > 1 ? (value > 2 ? 3 : 2) : 1) : 0;
}
