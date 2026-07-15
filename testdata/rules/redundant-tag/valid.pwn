main()
{
    new value = 1;
    new Float:converted = Float:value;
    new Float:literal = Float:1.0;
    new Float:source = 1.0;
    new untagged = _:source;
    new bool:flag = bool:value;
    return converted + literal + untagged + flag;
}
