Use(Float:value, bool:flag, raw)
{
    return raw;
}

Accept({Float,_}:value)
{
    return _:value;
}

Check(Float:f, bool:b, WEAPON:weapon, raw)
{
    new Float:structured[1][1];
    Use(f, b, raw);
    Accept(f);
    Accept(raw);
    Measure(f, weapon);
    Use(0, 0, 0);
    Measure(0, 0);
    Use(structured[0][0], b, raw);
}
