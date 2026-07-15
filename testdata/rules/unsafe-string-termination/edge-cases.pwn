#define RAW_COPY(%0,%1) memcpy(%0, %1, 0, 16 * 4)

Check(const source[])
{
    new macro_buffer[16];
    new conditional_buffer[16];
    RAW_COPY(macro_buffer, source);
    strlen(macro_buffer);
#if UNKNOWN_FEATURE
    memcpy(conditional_buffer, source, 0, 16 * 4);
    strlen(conditional_buffer);
#endif
}
