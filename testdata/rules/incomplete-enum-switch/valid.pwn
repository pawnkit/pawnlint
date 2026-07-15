enum PlayerState
{
    STATE_NONE,
    STATE_ALIVE,
    STATE_DEAD
}

AllStates(PlayerState:current)
{
    switch (current)
    {
        case STATE_NONE, STATE_ALIVE:
            return 0;
        case STATE_DEAD:
            return 1;
    }
    return 0;
}

DefaultState(PlayerState:current)
{
    switch (current)
    {
        case STATE_NONE:
            return 0;
        default:
            return 1;
    }
}

Untagged(value)
{
    switch (value)
    {
        case 0:
            return 0;
    }
    return 1;
}

enum Custom (+= 2)
{
    CUSTOM_A,
    CUSTOM_B
}

CustomSwitch(Custom:value)
{
    switch (value)
    {
        case CUSTOM_A:
            return 0;
    }
    return 1;
}
