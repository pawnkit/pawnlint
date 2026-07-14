after_return()
{
    return;
    new first;
    new second;
}

after_if()
{
    if (condition)
        return;
    else
        return;
    result = 1;
}

after_goto()
{
    goto done;
    skipped = 1;
done:
    return;
}

after_loop()
{
    while (true)
    {
    }
    result = 1;
}
