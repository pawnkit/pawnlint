conditional()
{
    new value;
#if defined UNKNOWN_FEATURE
    value = 1;
#endif
    return;
}

unreachable()
{
    new value;
    return;
    value = 1;
}

compound()
{
    new value;
    value += 1;
    value++, value = 2;
}

jump_read()
{
    new value;
    value = 1;
    goto used;
used:
    Use(value);
}

jump_exit()
{
    new value;
    value = 1;
    goto done;
done:
    return;
}
