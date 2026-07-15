ConditionalTernary(value)
{
    return value ? ConditionalTernary(value - 1) : 0;
}

SwitchBase(value)
{
    switch (value)
    {
        case 0:
            return 0;
        default:
            return SwitchBase(value - 1);
    }
}
