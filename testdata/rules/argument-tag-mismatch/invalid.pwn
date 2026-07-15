Use(Float:value, bool:flag, raw)
{
    return raw;
}

Check(Float:f, bool:b, raw)
{
    Use(raw, raw, f);
    Use(b, f, raw);
    Measure(raw, raw);
    Measure(f, b);
}
