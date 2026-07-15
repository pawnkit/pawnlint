#define NEGATIVE_PACKED(%0) (packed{%0} == -1)

Check()
{
    new packed[1 char];
#if UNKNOWN_FEATURE
    if (packed{0} == -1) return 1;
#endif
    return NEGATIVE_PACKED(0);
}
