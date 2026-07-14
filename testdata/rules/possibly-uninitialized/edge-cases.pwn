Use(value)
{
    return value;
}

conditional()
{
    new value;
#if defined UNKNOWN_FEATURE
    value = 1;
#endif
    Use(value);
}

unreachable()
{
    new value;
    return;
    Use(value);
}

for_header()
{
    for (new value; value != 2; value++)
    {
    }
}

jump_over_initializer()
{
    goto used;
    new value = 1;
used:
    Use(value);
}

switch_assigns(value)
{
    new result;
    switch (value)
    {
        case 1: result = 1;
        default: result = 2;
    }
    Use(result);
}
