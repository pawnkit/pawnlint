overwritten()
{
    new value;
    value = 1;
    value = 2;
    Use(value);
}

before_exit()
{
    new value;
    value = 1;
    return;
}

branches(bool:condition)
{
    new value;
    if (condition)
        value = 1;
    else
        value = 2;
}

repeated_loop()
{
    new value;
    while (Check())
    {
        value = 1;
    }
}
