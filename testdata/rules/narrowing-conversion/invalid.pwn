Check(value, condition)
{
    new packed[4 char];
    packed{0} = 256;
    packed{1} = -1;
    packed{2} = condition ? 0 : 300;
    packed /* storage */ {3} = value & 0x1FF;
}
