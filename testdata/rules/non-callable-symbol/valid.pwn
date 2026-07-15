native time();

main()
{
    printf("%d", time());

    new value = Add(1, 2);
    printf("%d", value);
}

Add(a, b)
{
    return a + b;
}

UnusedShadow()
{
    // A local sharing a native's name is fine as long as it is never
    // called; only calling it through this name is an error.
    new time;
    time = 5;
    printf("%d", time);
}
