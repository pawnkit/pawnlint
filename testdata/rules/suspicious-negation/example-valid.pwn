HasAdminFlag(flags)
{
    const FLAG_ADMIN = 1;
    return !(flags & FLAG_ADMIN);
}
