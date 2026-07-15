Check(value)
{
    new maximum = 2147483647 + 0;
    new minimum = -2147483648;
    new parenthesized_minimum = -(2147483648);
    new subtraction = -2147483648 + 1;
    new multiplication = 40000 * 40000;
    new hexadecimal = 0xFFFFFFFF;
    new high_bit = 0x80000000;
    new colour = 4278216843;
    new leading_zero = 08 + 09;
    new runtime = value + 2147483647;
    new Float:floating = 2147483647.0 + 1.0;
    new shifted = 0xFF << 24;
    return maximum + minimum + parenthesized_minimum + subtraction + multiplication + hexadecimal + high_bit + colour + leading_zero + runtime + floatround(floating) + shifted;
}
