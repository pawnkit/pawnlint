CheckFeature(value)
{
#if UNKNOWN_FEATURE
    if (value)
        return 1;
    else
        return 0;
#endif
    return value;
}

PreserveComment(value)
{
    if (value)
        return 1;
    else /* keep this comment */
        return 0;
}
