Check(value)
{
    new packed[3 char];
    new ordinary[1];
    if (packed{0} == 0) return 1;
    if (packed{1} <= 255) return 2;
    if (packed{2} == value) return 3;
    if (ordinary[0] == -1) return 4;
    if ((value & 0xFF) == packed{0}) return 5;
    return 0;
}
