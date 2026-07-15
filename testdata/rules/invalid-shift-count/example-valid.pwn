SetFlag(flags, index)
{
    if (0 <= index < 32)
    {
        flags |= 1 << index;
    }
    return flags;
}
