BuildLabel(count, value)
{
    new label[32];
    format(label, sizeof label, "Value: %d", value);
    for (new i; i < count; i++)
    {
        printf("%s", label);
    }
    return label[0];
}
