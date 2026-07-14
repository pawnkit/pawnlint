conditional()
{
#if defined UNKNOWN_FEATURE
    return;
#endif
    result = 1;
}

unknown_jump()
{
    goto missing;
    result = 1;
}

do_return()
{
    do
    {
        return;
    }
    while (false);
    result = 1;
}

false_loop()
{
    while (false)
    {
        result = 1;
    }
}
