DurationMs:GetDelay(Milliseconds:left, DurationMs:right, value)
{
    new DurationMs:initialized = left;
    initialized = right;
    initialized += left;
    if (initialized > right) return left;
    initialized = initialized + right;
    initialized = value;
    initialized = Milliseconds:value;
    return right;
}
