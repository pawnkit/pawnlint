ReadOnly(const &value)
{
    return value;
}

main()
{
    new throughConstReference = 1;
    ReadOnly(throughConstReference);

    new first = 1, second = 2;
    return first + second + throughConstReference;
}

#if UNKNOWN_FEATURE
main()
{
    new uncertain = 1;
    return uncertain;
}
#endif
