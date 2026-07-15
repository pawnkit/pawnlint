ConditionalBuild(first, second, third, fourth)
{
    #if defined UNKNOWN_FEATURE
        if (first && second && third && fourth)
        {
            return 1;
        }
    #endif
    return 0;
}

InactiveBuild(first, second, third, fourth)
{
    #if 0
        if (first || second || third || fourth)
        {
            return 1;
        }
    #endif
    return 0;
}

Independent(first, second, third, fourth, fifth, sixth)
{
    return (first && second && third) == (fourth || fifth || sixth);
}
