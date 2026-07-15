enum PlayerState
{
    STATE_NONE,
    STATE_ALIVE,
    STATE_DEAD
}

IsAlive(PlayerState:state)
{
	switch (state)
	{
		case STATE_ALIVE: return 1;
		case STATE_NONE: return 0;
		case STATE_DEAD: return 0;
    }
    return 0;
}
