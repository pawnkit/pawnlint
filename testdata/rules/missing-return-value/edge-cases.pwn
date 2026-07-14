uncertain(bool:condition)
{
#if defined UNKNOWN_FEATURE
    return 1;
#endif
}

unreachable_value()
{
    while (true)
    {
    }
    return 1;
}

jump_to_return()
{
    goto done;
done:
    return 1;
}

unknown_jump()
{
    if (condition)
        return 1;
    goto missing;
}
