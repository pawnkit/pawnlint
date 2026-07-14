main()
{
    new bool:repeat = true;
    SetTimer("Tick", 1000, repeat);
    (SetTimer("Tick", 1000, true));
}
