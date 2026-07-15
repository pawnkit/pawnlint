#define CALL_USE(%0) Use(%0, true, 0)

Use(Float:value, bool:flag, raw)
{
    return raw;
}

Check()
{
#if UNKNOWN_FEATURE
    Use(0, 0, Float:1);
#endif
    CALL_USE(0);
}
