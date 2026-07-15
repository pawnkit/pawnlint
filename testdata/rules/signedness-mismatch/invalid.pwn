Check()
{
    new packed[3 char];
    if (packed{0} == -1) return 1;
    if (-2 < packed{1}) return 2;
    if (packed{2} >= -10) return 3;
    return 0;
}
