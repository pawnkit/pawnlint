#include <core>

#if 0
main()
{
}
new inactiveLate;
#endif

#if defined UNKNOWN_FEATURE
main()
{
}
new uncertainLate;
#endif

new activeGlobal;

UseValue()
{
    #if defined UNKNOWN_LOCAL
    print("uncertain");
    new uncertainLocal;
    #endif
    return activeGlobal;
}
