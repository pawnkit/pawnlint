Use(value)
{
    return value;
}

SetValue(&value)
{
    value = 1;
}

initialized()
{
    new value = 1;
    Use(value);
}

both_branches(bool:condition)
{
    new value;
    if (condition)
        value = 1;
    else
        value = 2;
    Use(value);
}

do_assigns()
{
    new value;
    do
    {
        value = 1;
    }
    while (Check());
    Use(value);
}

ignored_storage()
{
    static stored;
    new values[4];
    Use(stored);
    Use(values);
}

written_by_reference()
{
    new value;
    SetValue(value);
    return value;
}

written_by_unknown_plugin()
{
    new value;
    PluginOutput(value);
    return value;
}
