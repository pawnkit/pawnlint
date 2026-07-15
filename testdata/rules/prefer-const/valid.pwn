ReadValue(value)
{
    return value;
}

MutateValue(&value)
{
    value++;
}

main()
{
    const existing = 1;
    new changed = 1;
    changed = 2;
    new incremented = 1;
    incremented++;
    new combined = 1;
    combined += 2;
    new missingInitializer;
    missingInitializer = 1;
    new unused = 1;
    new _intentionallyUnused = 1;
    new values[] = {1, 2};
    static stored = 1;
    new passedByReference = 1;
    MutateValue(passedByReference);
    new unresolvedArgument = 1;
    UnknownFunction(unresolvedArgument);
    return existing + changed + incremented + combined + missingInitializer + values[0] + stored;
}
