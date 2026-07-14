main()
{
    new value;
    switch (value)
    {
        case 1: return 1;
        case 2, 3: return 2;
        case 4 .. 6: return 3;
    }
}
