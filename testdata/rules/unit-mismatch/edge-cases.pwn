#define MIX_UNITS(%0,%1) ((%0) + (%1))

Milliseconds:Check(Milliseconds:milliseconds, Seconds:seconds)
{
#if UNKNOWN_FEATURE
    milliseconds = seconds;
#endif
    return MIX_UNITS(milliseconds, seconds);
}
