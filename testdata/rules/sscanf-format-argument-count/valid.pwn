main()
{
    new id, level, name[24], reason[128];

    sscanf("", "d", id);
    sscanf("", "dd", id, level);
    sscanf("", "s[24]d", name, level);
    sscanf("", "{s[4]}s[24]", name);
    sscanf("", "ds[128]", id, reason);
    sscanf("", "s[24]C(a)C()", name, level, id);
    sscanf("", "p<,>fff{fff}", id, level, id);
    sscanf("", "a<s[24]>[32]", name);
    sscanf("", "dD(0)", id, level);
    sscanf("", "u", id);

    new fmt[8] = "d";
    sscanf("", fmt, id);
}
