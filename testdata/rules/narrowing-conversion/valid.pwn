Check(value, condition)
{
    new packed[6 char];
    new ordinary[1];
    packed{0} = 0;
    packed{1} = 255;
    packed{2} = condition != 0;
    packed{3} = value & 0xFF;
    packed{4} = value >>> 24;
    packed{5} = value;
    ordinary[0] = 1000;
}
