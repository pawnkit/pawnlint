UseValues(first, second)
{
    return first + second;
}

WithDefault(value = (1, 2))
{
    return value;
}

main()
{
    new a = 1, b = 2, c = 3;
    new grouped = a * (b + c);
    new rightAssociative = a - (b - c);
    new explicitComparison = (a < b) < c;
    new commaValue = (a, b);
    UseValues((a, b), c);
    if ((a = b))
    {
        return -(a + b);
    }
    (Float:a);
    return grouped + rightAssociative + explicitComparison + commaValue;
}

#define SQUARE(%0) ((%0) * (%0))
