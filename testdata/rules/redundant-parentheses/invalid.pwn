UseValues(first, second)
{
    return first + second;
}

main()
{
    new a = 1, b = 2, c = 3;
    new simple = (a);
    new precedence = (a * b) + c;
    new tighter = a + (b * c);
    UseValues((a), b);
    if ((a))
    {
        return ((simple));
    }
    return simple + precedence + tighter;
}
