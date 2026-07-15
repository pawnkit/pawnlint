Check(const source[])
{
    new terminated[16];
    new overwritten[16];
    new binary[16];
    memcpy(terminated, source, 0, 16 * 4);
    terminated[15] = EOS;
    strlen(terminated);
    memcpy(overwritten, source, 0, 16 * 4);
    strmid(overwritten, source, 0, 15, sizeof(overwritten));
    strlen(overwritten);
    memcpy(binary, source, 0, 16 * 4);
}
