trivia(bool:condition)
{
    if (condition)
        result = 1;
    else
        /* same behavior */ result=1;
}

different_shape(bool:condition)
{
    if (condition)
        result = 1;
    else
    {
        result = 1;
    }
}

conditional(bool:condition)
{
    if (condition)
    {
#if defined UNKNOWN_FEATURE
        result = 1;
#endif
    }
    else
    {
#if defined UNKNOWN_FEATURE
        result = 1;
#endif
    }
}
