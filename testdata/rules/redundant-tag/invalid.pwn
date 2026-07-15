enum Colour
{
    Colour_Red
}

Float:KeepFloat(Float:value)
{
    return Float:value;
}

main()
{
    new Float:source = 1.0;
    new Float:copy = Float:source;
    new bool:ready = true;
    new bool:again = bool:ready;
    new Colour:colour = Colour_Red;
    new Colour:sameColour = Colour:colour;
    new Float:result = Float:KeepFloat(source);
    return copy + again + sameColour + result;
}
