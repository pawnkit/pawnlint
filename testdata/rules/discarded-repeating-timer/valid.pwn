new tickTimer;

main()
{
    tickTimer = SetTimer("Tick", 1000, true);
    SetTimer("OneShot", 1000, false);
    SetTimerEx("Delayed", 5000, false, "i", 1);
}
