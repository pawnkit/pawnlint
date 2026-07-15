Check(value, Float:ratio)
{
    if ((value & 255) == 255)
        return 1;
    if (value % 10 == 9)
        return 1;
    if ((value == 0) == 1)
        return 1;
    if (value > 2147483647)
        return 1;
    if (ratio > 2.0)
        return 1;
    if (1 < 2)
        return 1;
    if (0 <= value <= 10)
        return 1;
    if (5000 > value > 20000)
        return 1;
    return 0;
}

#define CHECK_RANGE(%0) (((%0) & 3) > 7)
