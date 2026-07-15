#define OPAQUE(x) x; x

#if UNKNOWN_FEATURE
#define CONDITIONAL(x) ((x) + (x))
#endif

main()
{
    return 1;
}
