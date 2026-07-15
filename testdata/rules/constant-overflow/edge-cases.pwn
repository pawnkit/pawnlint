#define WRAPPED_VALUE (2147483647 + 1)

Check()
{
#if UNKNOWN_FEATURE
    new conditional = 2147483647 + 1;
#endif
    return WRAPPED_VALUE;
}
