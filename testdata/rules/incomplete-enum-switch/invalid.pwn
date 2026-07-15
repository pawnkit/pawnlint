enum PlayerState
{
    STATE_NONE,
    STATE_ALIVE,
    STATE_DEAD
}

CheckState(PlayerState:current)
{
    switch (current)
    {
        case STATE_NONE:
            return 0;
    }
    return 1;
}

enum Code
{
    CODE_FIRST = 10,
    CODE_SECOND,
    CODE_THIRD
}

CheckCode(Code:code)
{
    switch (code)
    {
        case 10 .. 11:
            return 1;
    }
    return 0;
}
