main()
{
    new value;
    if (value)
        return;
    value = 1;
}

with_jump()
{
    goto done;
done:
    return;
}

with_break()
{
    while (true)
    {
        break;
    }
    return;
}
