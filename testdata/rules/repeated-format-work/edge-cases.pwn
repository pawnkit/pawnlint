#define WRITE_VALUE(%0,%1) format(%0, sizeof %0, "value %d", %1)

Check(limit, value)
{
    new macro_output[64];
    new conditional_output[64];
    for (new i; i < limit; i++) {
        WRITE_VALUE(macro_output, value);
    }
    Consume(macro_output);
#if UNKNOWN_FEATURE
    for (new i; i < limit; i++) {
        format(conditional_output, sizeof conditional_output, "value %d", value);
    }
#endif
    Consume(conditional_output);
}
