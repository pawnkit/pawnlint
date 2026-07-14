forward DeclaredOnly(value);
native NativeOnly(value);

stock Duplicate(value, value)
{
    return 1;
}

#if defined FEATURE
stock Conditional(value)
{
    return 1;
}
#endif
