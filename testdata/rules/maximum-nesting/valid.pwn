Allowed(value)
{
    while (value)
    {
        if (value > 1)
        {
            value--;
        }
    }
    return value;
}

ElseIf(value)
{
    if (value == 1)
    {
        return 1;
    }
    else if (value == 2)
    {
        return 2;
    }
    else if (value == 3)
    {
        return 3;
    }
    return 0;
}
