falls_through(bool:condition)
{
    if (condition)
        return 1;
}

bare_return(bool:condition)
{
    if (condition)
        return 1;
    return;
}

switch_path(value)
{
    switch (value)
    {
        case 1: return 1;
        default: result = 0;
    }
}
