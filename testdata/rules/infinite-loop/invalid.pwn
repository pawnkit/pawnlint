Forever()
{
    while (1)
    {
        print("forever");
    }
}

EmptyFor()
{
    for (;;)
    {
        print("forever");
    }
}

Invariant()
{
    new running = 1;
    while (running)
    {
        print("forever");
    }
}

NestedBreak(value)
{
    while (true)
    {
        switch (value)
        {
            case 0:
                break;
        }
    }
}

DoForever()
{
    do
    {
        print("forever");
    }
    while (true);
}
