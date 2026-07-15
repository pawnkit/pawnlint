Boundary(first, second, third)
{
    if (first)
    {
        first++;
    }
    if (second)
    {
        second++;
    }
    while (third)
    {
        third--;
    }
    return first + second + third;
}

SwitchValue(value)
{
    switch (value)
    {
        case 1: return 1;
        case 2: return 2;
        default: return 0;
    }
}
