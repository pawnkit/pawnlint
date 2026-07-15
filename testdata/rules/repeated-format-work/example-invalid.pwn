BuildLabel(count, value)
{
    new label[32];
    for (new i; i < count; i++)
    {
        format(label, sizeof label, "Value: %d", value);
    }
    return label[0];
}
