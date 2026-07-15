forward UnusedForward();

UnusedForward()
{
    return 1;
}

forward CalledAfterDefinition(value);

CalledAfterDefinition(value)
{
    return value;
}

main()
{
    return CalledAfterDefinition(1);
}
