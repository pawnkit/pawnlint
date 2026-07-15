BreakLoop()
{
    while (1)
    {
        break;
    }
}

ReturnLoop()
{
    for (;;)
    {
        return 1;
    }
}

ChangedCondition()
{
    new running = 1;
    while (running)
    {
        running = 0;
    }
}

UnknownCondition(value)
{
    while (value)
    {
        print("unknown");
    }

    while (IsReady())
    {
        print("unknown");
    }
}

GotoLoop()
{
    while (1)
    {
        goto done;
    }
done:
    return 1;
}

#define STOP_LOOP() break

MacroLoop()
{
    while (1)
    {
        STOP_LOOP();
    }
}
