BuildList(count, const item[])
{
    new output[128];
    for (new i; i < count; i++)
    {
        format(output, sizeof output, "%s%s", output, item);
    }
    return output[0];
}
