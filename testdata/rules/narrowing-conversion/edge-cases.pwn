#define STORE(%0,%1) packed{%0} = %1

Check()
{
    new packed[2 char];
#if UNKNOWN_FEATURE
    packed{0} = 256;
#endif
    STORE(1, 300);
}
