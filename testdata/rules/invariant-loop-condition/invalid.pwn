Check()
{
    new remaining = 10;
    while (remaining > 0)
    {
        print("waiting");
    }

    new limit = 5;
    for (new index = 0; limit < 10; index++)
    {
        print("limited");
    }

    new ready;
    do
    {
        print("checking");
    }
    while (!ready);

    new lower = 1;
    new upper = 10;
    while (lower < upper)
    {
        break;
    }
}
