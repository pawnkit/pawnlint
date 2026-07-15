Countdown(value)
{
    if (value <= 0)
    {
        return 0;
    }
    return Countdown(value - 1);
}
