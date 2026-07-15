Calculate(value)
{
    if (value > 42)
        return value * 60;

    SetTimer("Refresh", 2500, false);
    new Float:ratio = 3.5;

    switch (value)
    {
        case 7:
            return 1;
    }

    SendClientMessage(value, -1, "state");
    return 0;
}
