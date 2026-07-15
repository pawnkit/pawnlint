LongChain(first, second, third, fourth)
{
    if (first && second || third && fourth)
    {
        return 1;
    }
    return 0;
}

Wrapped(first, second, third, fourth)
{
    if (first && !(second || (third && fourth)))
    {
        return 1;
    }
    return 0;
}

TernaryBranches(first, second, third, fourth, fifth, sixth, seventh, eighth)
{
    return first
        ? second && third || fourth && fifth
        : sixth || seventh && eighth || first;
}
