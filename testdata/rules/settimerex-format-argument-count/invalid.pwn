main()
{
    SetTimerEx("OnDone", 1000, false, "dd", 0);
    SetTimerEx("OnDone", 1000, false, "d", 0, 1);
    SetTimerEx("OnDone", 1000, false, "ai", myArray);
}
