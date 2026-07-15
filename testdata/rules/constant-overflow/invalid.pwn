const MAX_VALUE = cellmax;

Check()
{
    new addition = 2147483647 + 1;
    new named = MAX_VALUE + 1;
    new subtraction = -2147483648 - 1;
    new multiplication = 50000 * 50000;
    new division = cellmin / -1;
    new negation = -cellmin;
    new literal = 4294967296;
    new hexadecimal = 0x100000000;
    new binary = 0b100000000000000000000000000000000;
    return addition + named + subtraction + multiplication + division + negation + literal + hexadecimal + binary;
}
