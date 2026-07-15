main()
{
    new id, level;

    // Unrecognized specifier letter: skipped rather than guessed at.
    sscanf("", "y", id);

    // Dynamic (non-literal) format string: skipped.
    new fmt[8] = "dd";
    sscanf("", fmt, id, level);

    // Named argument: skipped.
    sscanf("", "d", .id = id);
}
