#if 0
new inactiveFirst, inactiveSecond;
#endif

#if defined UNKNOWN_FEATURE
new uncertainFirst, uncertainSecond;
#endif

new active;

UseValue()
{
    #if 0
    new inactiveLocalFirst, inactiveLocalSecond;
    #endif
    return active;
}
