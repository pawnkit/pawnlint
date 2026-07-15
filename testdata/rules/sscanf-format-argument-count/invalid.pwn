main()
{
    new id, level, name[24];

    sscanf("", "dd", id);
    sscanf("", "d", id, level);
    sscanf("", "s[24]d", name);
    sscanf("", "{s[4]}s[24]", name, level);
}
