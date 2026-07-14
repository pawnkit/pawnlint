all_paths(bool:condition)
{
    if (condition)
        return 1;
    return 0;
}

no_value_result()
{
    if (condition)
        return;
}

value_or_loop(bool:condition)
{
    if (condition)
        return 1;
    while (true)
    {
    }
}
