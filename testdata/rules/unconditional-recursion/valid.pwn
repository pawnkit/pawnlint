WithBaseCase(value)
{
    if (value <= 0)
        return 0;
    return WithBaseCase(value - 1);
}

ConditionalCall(value)
{
    value && ConditionalCall(value - 1);
    return value;
}

MutualBaseA(value)
{
    if (value <= 0)
        return 0;
    return MutualBaseB(value - 1);
}

MutualBaseB(value)
{
    return MutualBaseA(value);
}

NonRecursiveLoop()
{
    while (1)
    {
        print("loop");
    }
}

LoopBeforeRecursion(value)
{
    while (value)
    {
        print("loop");
    }
    LoopBeforeRecursion(value);
}

#define RECURSE(%0) MacroRecursion(%0)

MacroRecursion(value)
{
    RECURSE(value);
}

DeferredRecursion()
{
    defer DeferredRecursion();
}
