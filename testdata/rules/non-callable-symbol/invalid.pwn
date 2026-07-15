native time();

ShadowedNative()
{
    new time;
    time = 5;
    printf("%d", time());
}

CalledParameter(count)
{
    return count();
}

CalledGlobal()
{
    return gTotal();
}

new gTotal;
