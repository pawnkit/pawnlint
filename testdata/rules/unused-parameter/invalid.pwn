stock Add(left, right)
{
    return left;
}

stock Empty(argc)
{
}

stock OtherPragmaScope(value)
{
}

stock UnrelatedPragma(value)
{
    #pragma unused OtherPragmaScope
}
