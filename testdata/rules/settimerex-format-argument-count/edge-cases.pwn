main()
{
    // Unrecognized specifier letter: skipped rather than guessed at.
    SetTimerEx("OnDone", 1000, false, "y", 0);

    // Dynamic (non-literal) specifier string: skipped.
    new fmt[8] = "dd";
    SetTimerEx("OnDone", 1000, false, fmt, 0);

    // Default (empty) specifier string with no extra arguments: fine.
    SetTimerEx("OnDone", 1000, false);
}
