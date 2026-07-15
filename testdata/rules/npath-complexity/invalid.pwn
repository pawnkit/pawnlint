ManyPaths(first, second, third)
{
    if (first)
    {
        first++;
    }
    if (second)
    {
        second++;
    }
    if (third)
    {
        third++;
    }
    while (first)
    {
        if (second)
        {
            break;
        }
        first--;
    }
    return first + second + third;
}

ExpressionPaths(first, second, third)
{
	first = second ? first : third;
	second = first ? second : third;
    if (first && second || third)
    {
        return first ? second : third;
    }
    return 0;
}
