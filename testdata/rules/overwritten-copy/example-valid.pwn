CopyName(const source[])
{
    new name[24];
    memcpy(name, source, 0, sizeof name * 4);
    return name[0];
}
