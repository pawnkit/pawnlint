ReadValue(value)
{
    return value;
}

main()
{
    new literal = 1;
    new expression = 2 + 3;
    new runtime = ReadValue(literal);
    new byValue = 4;
    ReadValue(byValue);
    return literal + expression + runtime + byValue;
}
