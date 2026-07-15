ComplexFlow(value, limit)
{
    if (value > 0 && limit > 0)
    {
        for (new index = 0; index < limit; index++)
        {
            value += index ? 1 : 2;
        }
    }
    switch (value)
    {
        case 1: return 1;
        case 2: return 2;
        default: return 0;
    }
}

BooleanPaths(first, second, third)
{
    return first || second || third ? 1 : 0;
}
