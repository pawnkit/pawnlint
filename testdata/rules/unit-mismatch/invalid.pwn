Milliseconds:GetDelay(Seconds:seconds)
{
    new Milliseconds:initialized = seconds;
    new Milliseconds:assigned;
    assigned = seconds;
    assigned += seconds;
    if (assigned < seconds) return seconds;
    assigned = assigned + seconds;
    assigned = true ? assigned : seconds;
    return seconds;
}
