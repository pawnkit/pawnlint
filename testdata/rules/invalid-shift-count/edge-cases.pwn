main()
{
#if defined FEATURE
    new conditional = 1 << 32;
#endif

    new unknown = 1 << SOME_MACRO;
}
