Direct()
{
    Direct();
}

ReturnDirect()
{
    return ReturnDirect();
}

BothBranches(value)
{
    if (value)
        BothBranches(value - 1);
    else
        BothBranches(value + 1);
}

MutualA()
{
    MutualB();
}

MutualB()
{
    MutualA();
}

AllSwitch(value)
{
    switch (value)
    {
        case 0:
            AllSwitch(1);
        default:
            AllSwitch(0);
    }
}

LeftShortCircuit(value)
{
    LeftShortCircuit(value - 1) && value;
}

TernaryCondition()
{
    return TernaryCondition() ? 1 : 0;
}
