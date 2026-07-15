Check(value)
{
    if ((value & 0xFF) > 255)
        return 1;
    if ((value & 3) <= 3)
        return 1;
    if (value % 10 >= 10)
        return 1;
    if (value % 10 < -9)
        return 1;
    if ((value == 0) == 2)
        return 1;
    if ((value == 0) != -1)
        return 1;
    if ((value >>> 8) < 0)
        return 1;
    if ((value ? (value & 3) : (value & 7)) > 7)
        return 1;
    return 0;
}
