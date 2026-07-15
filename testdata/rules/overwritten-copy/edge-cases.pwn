#define COPY(%0,%1) memcpy(%0, %1, 0, 16 * 4)

Check(const first[], const second[])
{
    new macro_buffer[16];
    new conditional_buffer[16];
    COPY(macro_buffer, first);
    COPY(macro_buffer, second);
#if UNKNOWN_FEATURE
    memcpy(conditional_buffer, first, 0, 16 * 4);
    memcpy(conditional_buffer, second, 0, 16 * 4);
#endif
}
