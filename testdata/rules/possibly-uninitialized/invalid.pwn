Use(value)
{
    return value;
}

direct_read()
{
    new value;
    Use(value);
}

one_branch(bool:condition)
{
    new value;
    if (condition)
        value = 1;
    Use(value);
}

optional_loop()
{
    new value;
    while (Check())
    {
        value = 1;
    }
    Use(value);
}

read_write()
{
    new value;
    value++;
}

self_initializer()
{
    new value = value;
}
