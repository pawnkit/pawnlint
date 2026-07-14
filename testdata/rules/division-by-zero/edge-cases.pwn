main()
{
#if defined FEATURE
    new conditional = 1 / 0;
#endif

#if 1
    new active = 1 / 0;
#else
    new inactive = 1 / 0;
#endif

    new unknown = 1 / SOME_MACRO;
}
