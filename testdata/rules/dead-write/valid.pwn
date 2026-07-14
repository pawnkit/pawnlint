used_after_write(bool:condition)
{
    new value;
    value = 1;
    Use(value);

    if (condition)
        value = 2;
    else
        value = 3;
    Use(value);
}

other_symbols(parameter)
{
    global_value = 1;
    parameter = 2;
}

do_condition()
{
    new value;
    do
    {
        value = 1;
    }
    while (value);
}
