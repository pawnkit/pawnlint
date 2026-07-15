#define OBJECT_LIKE First(); Second()

#if UNKNOWN_FEATURE
#define CONDITIONAL(x) First(x); Second(x)
#endif

main()
{
    return 1;
}
