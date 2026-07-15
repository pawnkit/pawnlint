Allowed(first, second, third)
{
    if (first && second || third)
    {
        return 1;
    }
    if (first & second | third ^ 1)
    {
        return 1;
    }
    return first && second;
}

Separate(first, second, third, fourth)
{
    return first && second ? third && fourth : 0;
}
