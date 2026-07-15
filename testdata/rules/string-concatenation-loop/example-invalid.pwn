BuildList(count, const item[])
{
    new output[128];
    for (new i; i < count; i++)
    {
        strcat(output, item, sizeof output);
    }
    return output[0];
}
