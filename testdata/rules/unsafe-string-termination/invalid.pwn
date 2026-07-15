Check(const source[])
{
    new first[16];
    new second[16];
    new conditional[16];
    new concatenated[16];
    memcpy(first, source, 0, 16 * 4);
    strlen(first);
    memcpy(second, source, 0, 16 * 4);
    strcmp(second, source);
    memcpy(conditional, source, 0, 16 * 4);
    if (strlen(source) > 0) {
        conditional[15] = EOS;
    }
    strlen(conditional);
    memcpy(concatenated, source, 0, 16 * 4);
    strcat(concatenated, "suffix");
}
